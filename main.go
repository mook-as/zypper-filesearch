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
	"strings"
	"text/tabwriter"

	"github.com/mook-as/zypper-filesearch/cmd"
	"github.com/mook-as/zypper-filesearch/cmd/filelist"
	"github.com/mook-as/zypper-filesearch/cmd/filesearch"
	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/itertools"
	"github.com/mook-as/zypper-filesearch/repository"
	"github.com/mook-as/zypper-filesearch/zypper"
)

func run(ctx context.Context) error {
	verbose := flag.Bool("verbose", false, "Enable debug logging")
	releaseVer := flag.String("releasever", "", "Set the value of $releasever in a .repo files")
	jsonFormat := flag.Bool("json", false, "Enable JSON output")
	xmlFormat := flag.Bool("xmlout", false, "Enable XML output")

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

	cmd.AddFlags()
	flag.Parse()

	var logOptions slog.HandlerOptions
	if *verbose {
		logOptions.Level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &logOptions)))

	// Make sure we can get the arch.
	if _, err := zypper.Arch(); err != nil {
		return err
	}

	db, err := database.New(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	repos, err := zypper.ListRepositories(ctx, *releaseVer)
	if err != nil {
		return err
	}
	if err := repository.Refresh(ctx, db, repos); err != nil {
		return err
	}

	results, err := cmd.Run(ctx, db, repos)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	if *jsonFormat {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			return err
		}
	} else if *xmlFormat {
		encoder := xml.NewEncoder(os.Stdout)
		encoder.Indent("", "  ")
		if err := encoder.Encode(results); err != nil {
			return err
		}
	} else {
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
				Name:  "Version",
				Value: func(result database.SearchResult) string { return result.Version },
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
