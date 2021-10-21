package secrets

import (
	"testing"
	"time"
)

// though not required, you can run a vault server locally for these tests:
// 		vault server -dev -dev-root-token-id="root"

// ensure these interfaces are implemented properly
var (
	_ SecretSymmetricKeyProvider = &VaultSecretProvider{}
	_ SecretStorage              = &VaultSecretProvider{}
)

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
