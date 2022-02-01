package data

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/infrahq/infra/internal/registry/models"
)

func TestEncryptionKeys(t *testing.T) {
	db := setup(t)

	k, err := CreateEncryptionKey(db, &models.EncryptionKey{
		Name:      "foo",
		Encrypted: []byte{0x00},
		Algorithm: "foo",
	})
	require.NoError(t, err)

	require.NotZero(t, k.KeyID)

	k2, err := GetEncryptionKey(db, ByEncryptionKeyID(k.KeyID))
	require.NoError(t, err)

	require.Equal(t, "foo", k2.Name)

	k3, err := GetEncryptionKey(db, ByName("foo"))
	require.NoError(t, err)

	require.Equal(t, k.ID, k3.ID)
}
