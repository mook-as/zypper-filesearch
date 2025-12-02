package zypper

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestListRepositories(t *testing.T) {
	_, err := ListRepositories(t.Context(), "")
	assert.NilError(t, err)
}
