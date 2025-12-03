// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

package repository

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/zypper"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

//go:embed testdata
var testdata embed.FS

func TestRefresh(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	db, err := database.NewTesting(t.Context())
	assert.NilError(t, err)

	subFS, err := fs.Sub(testdata, "testdata")
	assert.NilError(t, err)
	server := httptest.NewServer(http.FileServer(http.FS(subFS)))
	defer server.Close()

	// Check that we have no results before the refresh
	results, err := db.SearchFile(t.Context(), "*/zypper-filesearch/LICENSE*", "x86_64_v999", true)
	assert.NilError(t, err, "failed to search for files")
	assert.Check(t, cmp.Len(results, 0))

	err = Refresh(t.Context(), db, []*zypper.Repository{
		{
			Name:    "test",
			Type:    "rpm-md",
			Enabled: true,
			URL:     server.URL,
		},
	})
	assert.NilError(t, err)

	// Check that we found results after the refresh
	results, err = db.SearchFile(t.Context(), "*/zypper-filesearch/LICENSE*", "x86_64_v999", true)
	assert.NilError(t, err, "failed to search for files")
	assert.Assert(t, cmp.DeepEqual(results, []database.SearchResult{
		{
			Repository: "test",
			Package:    "zypper-filesearch",
			Arch:       "x86_64",
			Version:    "0.20251202T1523520800.235d9b57f3d8fbc2bc1856a34a088ba831bbae86-lp160.10.1",
			Path:       "/usr/share/licenses/zypper-filesearch/LICENSE.txt",
		},
	}))
}
