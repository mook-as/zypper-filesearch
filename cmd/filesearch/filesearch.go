// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

// Command `zypper-filesearch` searches for packages that contain files that
// match a given shell glob expression.
package filesearch

import (
	"context"
	"flag"
	"fmt"

	"github.com/mook-as/zypper-filesearch/cmd"
	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/zypper"
)

func New() cmd.CommandRunner {
	// No additional flags needed
	flag.Parse()

	return &command{}
}

type command struct {
}

func (c *command) AddFlags() {
}

// Run the `zypper-filesearch` command, including doing any argument parsing.
func (c *command) Run(ctx context.Context, db *database.Database, repos []*zypper.Repository) ([]database.SearchResult, error) {
	if flag.NArg() != 1 {
		return nil, fmt.Errorf("usage: zypper file-search [pattern]")
	}
	pattern := flag.Arg(0)

	arch, err := zypper.Arch()
	if err != nil {
		arch = ""
	}

	var results []database.SearchResult
	for _, arch := range []string{arch, ""} {
		for _, enabled := range []bool{true, false} {
			results, err = db.SearchFile(ctx, pattern, arch, enabled)
			if err != nil {
				return nil, err
			}
			if len(results) > 0 {
				break
			}
		}
	}

	return results, nil
}
