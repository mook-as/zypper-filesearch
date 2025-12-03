// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

// Command `zypper-filelist` lists files provided by a given package.
package filelist

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

// Run the `zypper-filelist` command, including doing any argument parsing.
func (c *command) Run(ctx context.Context, db *database.Database, repos []*zypper.Repository) ([]database.SearchResult, error) {
	if flag.NArg() == 0 {
		return nil, fmt.Errorf("usage: zypper file-list [pattern]")
	}

	arch, err := zypper.Arch()
	if err != nil {
		return nil, err
	}

	var results []database.SearchResult
	for _, arch := range []string{arch, ""} {
		results, err = db.ListPackage(ctx, arch, true, flag.Args()...)
		if err != nil {
			return nil, err
		}
		if len(results) > 0 {
			break
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	return results, nil
}
