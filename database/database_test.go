package database

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	db, err := New(t.Context())
	assert.NilError(t, err)
	assert.Check(t, db != nil, "no database")
}
