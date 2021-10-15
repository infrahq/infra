package secrets

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// test with https://github.com/nsmithuk/local-kms
// docker run -p 8380:8080 nsmithuk/local-kms

// ensure this interface is implemented properly
var _ SecretProvider = &AWSKMSSecretProvider{}

func TestKMSSealAndUnseal(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}

	k, err := NewAWSKMSSecretProvider(awskms)
	require.NoError(t, err)

	noRootKeyYet := ""

	key, err := k.GenerateDataKey("random", noRootKeyYet)
	require.NoError(t, err)
	require.NotEmpty(t, key.RootKeyID)

	key2, err := k.DecryptDataKey(key.RootKeyID, key.Encrypted)
	require.NoError(t, err)

	require.Equal(t, key, key2)

	encrypted, err := Seal(key, []byte("Your scientists were so preoccupied with whether they could, they didn’t stop to think if they should"))
	require.NoError(t, err)

	r, err := Unseal(key, encrypted)
	require.NoError(t, err)

	require.Equal(t, []byte("Your scientists were so preoccupied with whether they could, they didn’t stop to think if they should"), r)
}
