package secrets

import (
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
)

// you can run a vault server locally for these tests:
// 		vault server -dev -dev-root-token-id="root"

// ensure these interfaces are implemented properly
var _ SecretProvider = &VaultSecretProvider{}
var _ SecretStorage = &VaultSecretProvider{}

func TestVaultSaveAndLoadSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}
	addr := "http://localhost:8200"
	v, err := NewVaultSecretProvider(addr, "root", "")
	require.NoError(t, err)
	waitForVaultReady(t, v)

	err = v.SetSecret("foo", []byte("secret token"))
	require.NoError(t, err)

	r, err := v.GetSecret("foo")
	require.NoError(t, err)

	require.Equal(t, []byte("secret token"), r)
}

func TestVaultSealAndUnseal(t *testing.T) {
	if testing.Short() {
		t.Skip("test skipped in short mode")
		return
	}
	addr := "http://localhost:8200"
	v, err := NewVaultSecretProvider(addr, "root", "")
	require.NoError(t, err)

	waitForVaultReady(t, v)

	// make sure vault transit is configured
	_ = v.client.Sys().Mount("transit", &api.MountInput{Type: "transit"})

	noRootKeyYet := ""

	key, err := v.GenerateDataKey("random", noRootKeyYet)
	require.NoError(t, err)
	require.NotEmpty(t, key.RootKeyID)

	key2, err := v.DecryptDataKey(key.RootKeyID, key.Encrypted)
	require.NoError(t, err)

	require.Equal(t, key, key2)

	encrypted, err := Seal(key, []byte("garbo"))
	require.NoError(t, err)

	t.Log(string(encrypted))

	r, err := Unseal(key, encrypted)
	require.NoError(t, err)

	require.Equal(t, []byte("garbo"), r)
}

func waitForVaultReady(t *testing.T, v *VaultSecretProvider) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		h, _ := v.client.Sys().Health()
		if h != nil && h.Initialized && !h.Sealed {
			return // ready!
		}
		if time.Now().After(deadline) {
			t.Error("timeout waiting for vault to be ready")
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
