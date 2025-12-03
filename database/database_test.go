// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

package database

import (
	"os"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/mook-as/zypper-filesearch/zypper"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestNew(t *testing.T) {
	// Ensure we use a temporary directory for the database.
	cacheDir := t.TempDir()
	assert.NilError(t, os.Setenv("XDG_CACHE_HOME", cacheDir))
	xdg.Reload()

	// Create the database.
	db, err := New(t.Context())
	assert.NilError(t, err)
	assert.Check(t, db != nil, "no database")

	// Add some entries.
	repo := &zypper.Repository{
		Name:    "test",
		Type:    "rpm-md",
		Enabled: true,
		URL:     "http://fake-host.test",
	}
	lastModified := time.Unix(1231006505, 0).UTC()
	lastChecked := time.Unix(1231469665, 0).UTC()
	err = db.UpdateRepository(t.Context(), repo, lastChecked, lastModified, func(f func(pkgid, name, arch, version, file string) error) error {
		return f("pkg-id", "pkg-name", "avr32", "2:1.5-6", "/some/path")
	})
	assert.NilError(t, err)

	// Check that the modification times are correct
	actualChecked, actualModified, err := db.GetTimestamps(t.Context(), repo)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(lastModified, actualModified))
	assert.Check(t, cmp.Equal(lastChecked, actualChecked))

	// Check that we can find the file
	results, err := db.Search(t.Context(), "/some/path", "", true)
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(results, []SearchResult{
		{
			Repository: repo.Name,
			Package:    "pkg-name",
			Arch:       "avr32",
			Version:    "2:1.5-6",
			Path:       "/some/path",
		},
	}))

	// Check that the file can be written
	assert.NilError(t, db.Close())
	entries, err := os.ReadDir(cacheDir)
	assert.NilError(t, err)
	// It should just have the file, without WAL/journal.
	assert.Check(t, cmp.Len(entries, 1))

	// Check that the data was persisted
	db, err = New(t.Context())
	assert.NilError(t, err)
	assert.Assert(t, db != nil, "no database")
	results, err = db.Search(t.Context(), "/some/path", "", true)
	assert.NilError(t, err)
	assert.Check(t, cmp.Len(results, 1))

	assert.NilError(t, db.Close())
}
