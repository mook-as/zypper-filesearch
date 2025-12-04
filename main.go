// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/mook-as/zypper-filesearch/cmd"
	"github.com/mook-as/zypper-filesearch/cmd/filelist"
	"github.com/mook-as/zypper-filesearch/cmd/filesearch"
	"github.com/mook-as/zypper-filesearch/config"
	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/itertools"
	"github.com/mook-as/zypper-filesearch/repository"
	"github.com/mook-as/zypper-filesearch/zypper"
)

func run(ctx context.Context) error {
	var cmd cmd.CommandRunner

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if strings.HasSuffix(exe, "zypper-file-list") {
		cmd = filelist.New()
	} else {
		cmd = filesearch.New()
	}

	config.AddFlags()
	cmd.AddFlags()
	flag.Parse()

	cfg, err := config.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read configuration: %w", err)
	}

	var logOptions slog.HandlerOptions
	if cfg.Verbose {
		logOptions.Level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &logOptions)))

	slog.DebugContext(ctx, "Initial setup complete")
	// Make sure we can get the arch.
	if _, err := zypper.Arch(); err != nil {
		return err
	}

	slog.DebugContext(ctx, "Opening database")
	db, err := database.New(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()
	slog.DebugContext(ctx, "Database opened")

	repos, err := zypper.ListRepositories(ctx, cfg.ReleaseVer)
	if err != nil {
		return err
	}
	if cfg.Enabled {
		// Filter out disabled repositories
		repos = slices.DeleteFunc(repos, func(r *zypper.Repository) bool {
			return !r.Enabled
		})
	}
	if err := repository.Refresh(ctx, db, repos); err != nil {
		return err
	}

	results, err := cmd.Run(ctx, cfg, db, repos)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	switch cfg.Format {
	case config.OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			return err
		}
	case config.OutputFormatXML:
		encoder := xml.NewEncoder(os.Stdout)
		encoder.Indent("", "  ")
		if err := encoder.Encode(results); err != nil {
			return err
		}
	case config.OutputFormatHuman:
		type field struct {
			Name  string
			Value func(result database.SearchResult) string
		}
		writer := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fields := []field{
			{
				Name:  "Repository",
				Value: func(result database.SearchResult) string { return result.Repository },
			},
			{
				Name:  "Package",
				Value: func(result database.SearchResult) string { return result.Package },
			},
			{
				Name: "Version",
				Value: func(result database.SearchResult) string {
					version := result.Version
					if result.Epoch != "" && result.Epoch != "0" {
						version = result.Epoch + ":" + version
					}
					if result.Release != "" {
						version += "-" + result.Release
					}
					return version
				},
			},
			{
				Name:  "Arch",
				Value: func(result database.SearchResult) string { return result.Arch },
			},
			{
				Name:  "File",
				Value: func(result database.SearchResult) string { return result.Path },
			},
		}
		writeLine := func(f func(field) string) error {
			_, err := fmt.Fprintf(writer, "%s\n", strings.Join(itertools.Map(fields, f), "\t"))
			return err
		}

		if err := writeLine(func(f field) string { return f.Name }); err != nil {
			return err
		}
		if err := writeLine(func(f field) string { return "---" }); err != nil {
			return err
		}
		for _, result := range results {
			if err := writeLine(func(f field) string { return f.Value(result) }); err != nil {
				return err
			}
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	err := run(context.Background())
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
