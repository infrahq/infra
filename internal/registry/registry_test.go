package registry

import (
	"testing"

	"github.com/infrahq/infra/secrets"
	"github.com/stretchr/testify/assert"
)

func TestReplaceSecretTemplatesRemovesPlaintextTemplates(t *testing.T) {
	r := &Registry{secrets: make(map[string]secrets.SecretStorage)}

	f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
	r.secrets["plaintext"] = f

	// in reality, the user doesn't need to be a secret, just testing multiple secrets in the DSN
	dsnTemplates := "host=host.docker.internal user={{plaintext:postgres}} password={{plaintext:password}} dbname=postgres port=5432"

	processed, err := r.ReplaceSecretTemplates(dsnTemplates)
	if err != nil {
		t.Fatalf(err.Error())
	}

	assert.Equal(t, "host=host.docker.internal user=postgres password=password dbname=postgres port=5432", processed)
}
