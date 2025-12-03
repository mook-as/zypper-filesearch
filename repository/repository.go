// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

// Package repository contains the code to keep the database up to date.
package repository

import (
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/zypper"
	"golang.org/x/sync/errgroup"
)

type fetchType func(ctx context.Context, name, kind string, parts ...string) (io.ReadCloser, error)

func fetchHttp(ctx context.Context, name, kind string, urlParts ...string) (io.ReadCloser, error) {
	finalURL, err := url.JoinPath(urlParts[0], urlParts[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s URL: %w", kind, err)
	}
	slog.DebugContext(ctx, "Fetching file", "kind", kind, "url", finalURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, finalURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to construct HTTP request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s from %s: %w", kind, name, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch %s from %s: status code %d (%s)", kind, name, resp.StatusCode, resp.Status)
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("failed to fetch %s from %s: no body", kind, name)
	}

	return resp.Body, nil
}

func updateRepository(ctx context.Context, db *database.Database, repo *zypper.Repository, fetch fetchType) error {
	if repo.Type != "rpm-md" {
		slog.WarnContext(ctx,
			"Skipping repository of unknown type",
			"repository", repo.Name, "type", repo.Type)
		return nil
	}
	lastUpdated, lastModified, err := db.GetTimestamps(ctx, repo)
	if err != nil {
		return err
	}
	if lastUpdated.Add(time.Hour).After(time.Now()) {
		slog.DebugContext(ctx,
			"Repository does not require update",
			"repository", repo.Name, "last update", lastUpdated.Local())
		return nil
	}
	slog.DebugContext(ctx, "Updating repository",
		"repository", repo.Name, "url", repo.URL, "last update", lastUpdated.Local())
	updateStartTime := time.Now().UTC()

	mdBody, err := fetch(ctx, repo.Name, "repomd.xml", repo.URL, "repodata", "repomd.xml")
	if err != nil {
		if !repo.Enabled {
			return nil // Ignore errors from disabled repositories
		}
		return err
	}
	defer func() {
		_ = mdBody.Close()
	}()
	type repomdData struct {
		Type     string `xml:"type,attr"`
		Checksum struct {
			Type  string `xml:"type,attr"`
			Value string `xml:",chardata"`
		} `xml:"checksum"`
		Location struct {
			Href string `xml:"href,attr"`
		} `xml:"location"`
		Timestamp int64 `xml:"timestamp"`
		Size      int   `xml:"size"`
	}
	var repomd struct {
		Data []repomdData `xml:"data"`
	}
	if err := xml.NewDecoder(mdBody).Decode(&repomd); err != nil {
		return fmt.Errorf("failed to parse repomd.xml from %s: %w", repo.Name, err)
	}
	_ = mdBody.Close()

	fileListIndex := slices.IndexFunc(repomd.Data, func(d repomdData) bool {
		return d.Type == "filelists"
	})
	if fileListIndex < 0 {
		return fmt.Errorf("repository %s does not have file lists", repo.Name)
	}
	timestamp := time.Unix(repomd.Data[fileListIndex].Timestamp, 0).UTC()
	if timestamp.Equal(lastModified) {
		slog.DebugContext(ctx, "File list has not changed",
			"repository", repo.Name, "last update", lastModified.Local())
		return nil
	}

	fileListBody, err := fetch(ctx,
		repo.Name, "filelists.xml", repo.URL, repomd.Data[fileListIndex].Location.Href)
	if err != nil {
		if !repo.Enabled {
			return nil // Ignore errors from disabled repositories
		}
		return err
	}
	defer func() {
		_ = fileListBody.Close()
	}()

	var hasher hash.Hash
	switch repomd.Data[fileListIndex].Checksum.Type {
	case "sha512":
		hasher = sha512.New()
	}
	fileListReader := fileListBody.(io.Reader)
	if hasher != nil {
		fileListReader = io.TeeReader(fileListBody, hasher)
	}

	switch path.Ext(repomd.Data[fileListIndex].Location.Href) {
	case ".gz":
		fileListReader, err = gzip.NewReader(fileListReader)
	case ".zst":
		fileListReader, err = zstd.NewReader(fileListReader)
	}
	if err != nil {
		return fmt.Errorf("failed to decompress filelists.xml: %w", err)
	}

	var data struct {
		Package []*struct {
			PkgId   string `xml:"pkgid,attr"`
			Name    string `xml:"name,attr"`
			Arch    string `xml:"arch,attr"`
			Version struct {
				Epoch   string `xml:"epoch,attr"`
				Version string `xml:"ver,attr"`
				Release string `xml:"rel,attr"`
			} `xml:"version"`
			Files []*struct {
				Type string `xml:"type,attr"`
				Path string `xml:",chardata"`
			} `xml:"file"`
		} `xml:"package"`
	}

	if err := xml.NewDecoder(fileListReader).Decode(&data); err != nil {
		return fmt.Errorf("failed to parse filelists.xml from %s: %w", repo.Name, err)
	}

	if hasher != nil {
		sum := fmt.Sprintf("%02x", hasher.Sum(nil))
		if sum != repomd.Data[fileListIndex].Checksum.Value {
			slog.WarnContext(ctx, "File list has incorrect checksum",
				"repository", repo.Name,
				"expected", repomd.Data[fileListIndex].Checksum.Value,
				"actual", sum)
		}
	}

	err = db.UpdateRepository(ctx, repo, updateStartTime, timestamp, func(add func(pkgid, name, arch, version, file string) error) error {
		for _, pkg := range data.Package {
			var version string
			if pkg.Version.Epoch == "" || pkg.Version.Epoch == "0" {
				version = fmt.Sprintf("%s-%s", pkg.Version.Version, pkg.Version.Release)
			} else {
				version = fmt.Sprintf("%s:%s-%s", pkg.Version.Epoch, pkg.Version.Version, pkg.Version.Release)
			}
			for _, file := range pkg.Files {
				if file.Type == "dir" {
					continue
				}
				if !filepath.IsAbs(file.Path) {
					continue
				}
				if err := add(pkg.PkgId, pkg.Name, pkg.Arch, version, file.Path); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func Refresh(ctx context.Context, db *database.Database, repos []*zypper.Repository) error {
	wg, ctx := errgroup.WithContext(ctx)
	for _, repo := range repos {
		wg.Go(func() error {
			if !strings.HasPrefix(repo.URL, "http://") && !strings.HasPrefix(repo.URL, "https://") {
				slog.WarnContext(ctx, "Skipping non-HTTP repository",
					"repository", repo.Name, "url", repo.URL)
				return nil
			}
			return updateRepository(ctx, db, repo, fetchHttp)
		})
	}
	return wg.Wait()
}
