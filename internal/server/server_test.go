package server

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

	r := &Server{options: Options{}, secrets: make(map[string]secrets.SecretStorage)}

	f := secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{})
	r.secrets["plaintext"] = f

	url, err := r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Empty(t, url)

	r.options.DBHost = "localhost"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost", url)

	r.options.DBPort = 5432

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)
	require.Equal(t, "host=localhost port=5432", url)

	r.options.DBUser = "user"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost user=user port=5432", url)

	r.options.DBPassword = "plaintext:secret"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost user=user password=secret port=5432", url)

	r.options.DBName = "postgres"

	url, err = r.getPostgresConnectionString()
	require.NoError(t, err)

	require.Equal(t, "host=localhost user=user password=secret port=5432 dbname=postgres", url)
}
