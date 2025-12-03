// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

// Package cmd defines the interface all commands must implement
package cmd

import (
	"context"

	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/zypper"
)

type CommandRunner interface {
	// Add any flags this command requires.
	AddFlags()
	// Run the command, with the given options.
	Run(
		context.Context,
		*database.Database,
		[]*zypper.Repository,
	) ([]database.SearchResult, error)
}
