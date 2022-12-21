/*
Package patch provides helper functions for patching static variables in tests.

Ideally we would not use exported static package-level variables, but as long as
we have them, we need to patch them in tests.
*/
package patch

import (
	"path/filepath"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data/encrypt"
	"github.com/infrahq/infra/internal/server/models"
)

type TestingT interface {
	assert.TestingT
	Helper()
	TempDir() string
	Cleanup(func())
}

// ModelsSymmetricKey sets model.ModelsSymmetricKey to a random key for the lifetime of the test.
// This function modifies global state, it must not be used with t.Parallel.
func ModelsSymmetricKey(t TestingT) {
	t.Helper()

	rootKeyPath := filepath.Join(t.TempDir(), "db_at_rest")
	assert.NilError(t, encrypt.CreateRootKey(rootKeyPath))

	key, err := encrypt.CreateDataKey(rootKeyPath)
	assert.NilError(t, err)

	models.SymmetricKey = key
	t.Cleanup(func() {
		models.SymmetricKey = nil
	})
}
