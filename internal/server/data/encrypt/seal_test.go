package encrypt

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSealAndUnseal(t *testing.T) {
	tmp := t.TempDir()
	rootKeyPath := filepath.Join(tmp, "root-key")
	assert.NilError(t, CreateRootKey(rootKeyPath))

	key, err := CreateDataKey(rootKeyPath)
	assert.NilError(t, err)

	secretMessage := "This is the message"
	encrypted, err := Seal(key, []byte(secretMessage))
	assert.NilError(t, err)

	unsealed, err := Unseal(key, encrypted)
	assert.NilError(t, err)
	assert.Equal(t, secretMessage, string(unsealed))
}

func TestDecryptDataKey(t *testing.T) {
	tmp := t.TempDir()
	rootKeyPath := filepath.Join(tmp, "root-key")
	assert.NilError(t, CreateRootKey(rootKeyPath))

	dataKey, err := CreateDataKey(rootKeyPath)
	assert.NilError(t, err)

	actual, err := DecryptDataKey(rootKeyPath, dataKey.Encrypted)
	assert.NilError(t, err)

	assert.DeepEqual(t, actual.unencrypted, dataKey.unencrypted)
}
