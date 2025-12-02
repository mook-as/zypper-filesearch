package repository

import (
	"testing"

	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/zypper"
	"gotest.tools/v3/assert"
)

func TestRefresh(t *testing.T) {
	db, err := database.NewTesting(t.Context())
	assert.NilError(t, err)
	// TODO: Set up a RPM repository for this project, and use that for the test
	err = Refresh(t.Context(), db, []*zypper.Repository{
		{
			Name: "openh264",
			Type: "rpm-md",
			Enabled: true,
			URL: "http://codecs.opensuse.org/openh264/openSUSE_Leap_16/",
		},
	})
	assert.NilError(t, err)
}