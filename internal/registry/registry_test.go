package registry

import (
	"testing"

	"github.com/infrahq/infra/secrets"
	"github.com/stretchr/testify/assert"
)

func TestGetPostgresConnectionURL(t *testing.T) {
	pg := PostgresOptions{}
	opts := Options{PostgresOptions: pg}
	r := &Registry{options: opts, secrets: make(map[string]secrets.SecretStorage)}

	f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
	r.secrets["plaintext"] = f

	url, err := r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	assert.Empty(t, url)

	r.options.PostgresOptions.PostgresHost = "localhost"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "host=localhost", url)

	r.options.PostgresOptions.PostgresPort = 5432

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "host=localhost port=5432", url)

	r.options.PostgresOptions.PostgresUser = "user"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "host=localhost user=user port=5432", url)

	r.options.PostgresOptions.PostgresPassword = "secret"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "host=localhost user=user password=secret port=5432", url)

	r.options.PostgresOptions.PostgresDBName = "postgres"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "host=localhost user=user password=secret port=5432 dbname=postgres", url)
}
