/*Package patch provides helper functions for patching static variables in tests.

Ideally we would not use exported static package-level variables, but as long as
we have them, we need to patch them in tests.
*/
package patch

import (
	"github.com/infrahq/secrets"
	"gotest.tools/v3/assert"

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
	sp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{Path: t.TempDir()})

	rootKey := "db_at_rest"
	kp := secrets.NewNativeKeyProvider(sp)
	key, err := kp.GenerateDataKey(rootKey)
	assert.NilError(t, err)

	models.SymmetricKey = key
	t.Cleanup(func() {
		models.SymmetricKey = nil
	})
}
