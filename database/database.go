// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

// Package database contains the wrappers for operating with the database.
package database

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/adrg/xdg"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mook-as/zypper-filesearch/itertools"
	"github.com/mook-as/zypper-filesearch/zypper"
)

const (
	applicationId = int32(0x11668798)
	userVersion   = int32(2)
)

type Database struct {
	db *sql.DB
}

func New(ctx context.Context) (*Database, error) {
	filePath, err := xdg.CacheFile("zypper-filesearch.db")
	if err != nil {
		return nil, fmt.Errorf("failed to determine database file path: %w", err)
	}

	db, err := sql.Open("sqlite3", "file:"+filePath+"?mode=rwc&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(1)

	d := &Database{
		db: db,
	}

	if err := d.initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return d, nil
}

// Create an empty in-memory database for testing.
func NewTesting(ctx context.Context) (*Database, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{
		db: db,
	}

	if err := d.initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return d, nil
}

// initialize the database, performing migrations as necessary.
func (d *Database) initialize(ctx context.Context) error {
	var version int32
	_, err := d.db.ExecContext(ctx, fmt.Sprintf("PRAGMA application_id = %d", applicationId))
	if err != nil {
		return fmt.Errorf("failed to set database application id: %w", err)
	}

	for _, stmt := range []string{
		"PRAGMA auto_vacuum = 1",
		"PRAGMA encoding = 'UTF-8'",
		"PRAGMA foreign_keys = 1",
		"PRAGMA journal_mode = WAL",
		"PRAGMA recursive_triggers = 1",
	} {
		if _, err := d.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute pragma %q: %w", stmt, err)
		}
	}

	err = d.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to get database version: %w", err)
	}
	if version == userVersion {
		// This is a valid database
		return nil
	}

	// The database may have incompatible data; because this is only used for
	// a cache, we can just drop everything.
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS repositories`,
		`CREATE TABLE repositories (` +
			`id INTEGER PRIMARY KEY AUTOINCREMENT, ` +
			`alias TEXT, ` +
			`name TEXT, ` +
			`url TEXT UNIQUE ON CONFLICT ABORT, ` +
			`type TEXT, ` +
			`enabled BOOLEAN, ` +
			`lastChecked DATE, ` +
			`lastModified DATE` +
			`)`,
		`DROP TABLE IF EXISTS packages`,
		`CREATE TABLE packages (` +
			`repository INTEGER REFERENCES repositories(id) ON DELETE CASCADE, ` +
			`id INTEGER PRIMARY KEY AUTOINCREMENT, ` +
			`pkgid TEXT UNIQUE, ` +
			`name TEXT, ` +
			`arch TEXT, ` +
			`version TEXT, ` +
			`UNIQUE (repository, name, arch, version))`,
		`DROP TABLE IF EXISTS files`,
		`CREATE TABLE files (` +
			`pkgid TEXT REFERENCES packages(id) ON DELETE CASCADE, ` +
			`file TEXT,
			PRIMARY KEY (pkgid, file))`,
	} {
		if _, err := d.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to initialize database: %q: %w", stmt, err)
		}
	}

	_, err = d.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", userVersion))
	if err != nil {
		return fmt.Errorf("failed to set database version: %w", err)
	}
	return nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

// Look up when the given repository was last checked, and last modified.
func (d *Database) GetTimestamps(ctx context.Context, repo *zypper.Repository) (time.Time, time.Time, error) {
	var lastChecked, lastModified time.Time
	err := d.db.QueryRowContext(ctx, "SELECT lastChecked, lastModified FROM repositories WHERE url = ?", repo.URL).Scan(&lastChecked, &lastModified)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return lastChecked.UTC(), lastModified.UTC(), nil
}

// Update a given repository; all updates should be done within the passed-in
// function, as that will be used to establish a transaction.  It will be passed
// a function which can update one file at a time.
func (d *Database) UpdateRepository(ctx context.Context, repo *zypper.Repository, lastChecked, lastModified time.Time, cb func(pkg func(pkgid, name, arch, version string) error, file func(pkgid, file string) error) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// If we return before the commit, do a rollback.  This is a no-op if we have
	// already committed.
	defer func() {
		_ = tx.Rollback()
	}()

	// This drops any existing data for the repository because we enable the
	// recursive_triggers pragma, which per https://www.sqlite.org/lang_conflict.html:
	// > When the REPLACE conflict resolution strategy deletes rows in order to
	// > satisfy a constraint, delete triggers fire if and only if recursive
	// > triggers are enabled.
	result, err := tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO repositories `+
			`(alias, name, url, type, enabled, lastChecked, lastModified) `+
			`VALUES (?, ?, ?, ?, ?, ?, ?)`,
		repo.Alias, repo.Name, repo.URL, repo.Type, repo.Enabled, lastChecked, lastModified)
	if err != nil {
		return fmt.Errorf("failed to update repository %s: %w", repo.Name, err)
	}

	rowid, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last inserted id when updating repository %s: %w", repo.Name, err)
	}

	pkgStmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO packages (repository, pkgid, name, arch, version) `+
			`VALUES(?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	fileStmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO files (pkgid, file) `+
			`VALUES ((SELECT id FROM packages WHERE packages.pkgid = ?), ?)`)
	if err != nil {
		return err
	}

	err = cb(func(pkgid, name, arch, version string) error {
		_, err := pkgStmt.ExecContext(ctx, rowid, pkgid, name, arch, version)
		if err != nil {
			return fmt.Errorf("failed to update package: %w", err)
		}
		return nil
	}, func(pkgid, file string) error {
		_, err := fileStmt.ExecContext(ctx, pkgid, file)
		if err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error commiting update of repository %s: %w", repo.Name, err)
	}
	return nil
}

type SearchResult struct {
	XMLName    xml.Name `json:"-" xml:"result"`
	Repository string   `json:"repository" xml:"repository,attr"`
	Package    string   `json:"package" xml:"package,attr"`
	Arch       string   `json:"arch" xml:"arch,attr"`
	Version    string   `json:"version" xml:"versionm,attr"`
	Path       string   `json:"path" xml:"path,attr"`
}

func (d *Database) buildRepoFilter(repos []*zypper.Repository) (string, []any) {
	query := fmt.Sprintf("(%s)", strings.Join(itertools.Map(repos, func(r *zypper.Repository) string { return "?" }), ", "))
	args := itertools.Map(repos, func(r *zypper.Repository) any { return r.URL })
	return query, args
}

// Search for a file: Given a file path as a glob pattern, return packages with
// matching files.
func (d *Database) SearchFile(ctx context.Context, repos []*zypper.Repository, path, arch string) ([]SearchResult, error) {
	repoQuery, repoArgs := d.buildRepoFilter(repos)

	query := `SELECT repositories.name, packages.name, packages.arch, packages.version, files.file ` +
		`FROM packages INNER JOIN repositories ON packages.repository == repositories.id ` +
		`INNER JOIN files ON packages.id == files.pkgid ` +
		`WHERE files.file GLOB ? AND repositories.url IN ` + repoQuery
	if arch != "" {
		query += fmt.Sprintf(` AND (packages.arch == 'noarch' OR '%s' LIKE packages.arch || '%%' )`, arch)
	}

	rows, err := d.db.QueryContext(ctx, query, slices.Concat([]any{path}, repoArgs)...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.Repository, &result.Package, &result.Arch, &result.Version, &result.Path); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading query results: %w", err)
	}

	return results, nil
}

func (d *Database) ListPackage(ctx context.Context, repos []*zypper.Repository, arch string, terms ...string) ([]SearchResult, error) {
	repoQuery, repoArgs := d.buildRepoFilter(repos)

	pkgQuery := `SELECT packages.id ` +
		`FROM packages INNER JOIN repositories ON packages.repository == repositories.id ` +
		`WHERE packages.name == ? AND (? == '' OR packages.version == ?) AND repositories.url IN ` +
		repoQuery
	if arch != "" {
		pkgQuery += fmt.Sprintf(` AND (packages.arch == 'noarch' OR '%s' LIKE packages.arch || '%%' )`, arch)
	}
	pkgStmt, err := d.db.Prepare(pkgQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %q", err)
	}
	var pkgIds []int
	for _, term := range terms {
		term = strings.TrimSuffix(term, "-")
		// `pkg` may be `pkg-version` or `pkg-version-build`
		candidates := []int{-1}
		if i := strings.LastIndex(term, "-"); i > -1 {
			candidates = append(candidates, i)
			if j := strings.LastIndex(term[:i], "-"); j > -1 {
				candidates = append(candidates, j)
			}
		}

		found := false
		for _, i := range candidates {
			pkg := term
			version := ""
			if i > -1 {
				pkg = term[:i]
				version = term[i+1:]
			}
			rows, err := pkgStmt.QueryContext(ctx, slices.Concat([]any{pkg, version, version}, repoArgs)...)
			if err != nil {
				return nil, fmt.Errorf("failed to query package %s: %w", pkg, err)
			}
			defer func() {
				_ = rows.Close()
			}()
			for rows.Next() {
				found = true
				var pkgId int
				if err := rows.Scan(&pkgId); err != nil {
					return nil, fmt.Errorf("failed to get package %s (%s) id: %w", pkg, version, err)
				}
				pkgIds = append(pkgIds, pkgId)
			}
			_ = rows.Close()
			if found {
				break
			}
		}
		if !found {
			slog.ErrorContext(ctx, "package not found", "package", term)
		}
	}

	query := `SELECT repositories.name, packages.name, packages.arch, packages.version, files.file ` +
		`FROM packages INNER JOIN repositories ON packages.repository == repositories.id ` +
		`INNER JOIN files ON packages.id == files.pkgid ` +
		`WHERE packages.id IN ` +
		fmt.Sprintf("(%s)", strings.Join(itertools.Map(pkgIds, func(s int) string { return "?" }), ", "))
	rows, err := d.db.QueryContext(ctx, query, itertools.Map(pkgIds, func(s int) any { return s })...)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.Repository, &result.Package, &result.Arch, &result.Version, &result.Path); err != nil {
			return nil, fmt.Errorf("failed to read package list: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}
