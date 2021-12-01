package registry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/secrets"
)

func setupLogging(t *testing.T) {
	logging.L = zaptest.NewLogger(t)
	logging.S = logging.L.Sugar()
}

func TestGetPostgresConnectionURL(t *testing.T) {
	setupLogging(t)

	pg := PostgresOptions{}
	opts := Options{PostgresOptions: pg}
	r := &Registry{options: opts, secrets: make(map[string]secrets.SecretStorage)}

	f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
	r.secrets["plaintext"] = f

	url, err := r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	require.Empty(t, url)

	r.options.PostgresOptions.PostgresHost = "localhost"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "host=localhost", url)

	r.options.PostgresOptions.PostgresPort = 5432

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "host=localhost port=5432", url)

	r.options.PostgresOptions.PostgresUser = "user"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "host=localhost user=user port=5432", url)

	r.options.PostgresOptions.PostgresPassword = "plaintext:secret"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "host=localhost user=user password=secret port=5432", url)

	r.options.PostgresOptions.PostgresDBName = "postgres"

	url, err = r.getPostgresConnectionString()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "host=localhost user=user password=secret port=5432 dbname=postgres", url)
}
