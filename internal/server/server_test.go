package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/infrahq/infra/internal/config"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
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

func TestSetupRequired(t *testing.T) {
	db := setupDB(t)
	s := Server{db: db}

	// cases where setup is enabled
	cases := map[string]Options{
		"EnableSetup": {
			EnableSetup: true,
		},
		"NoImportProviders": {
			EnableSetup: true,
			Import: &config.Config{
				Providers: []config.Provider{},
			},
		},
		"NoImportGrants": {
			EnableSetup: true,
			Import: &config.Config{
				Grants: []config.Grant{},
			},
		},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			s.options = options
			require.True(t, s.setupRequired())
		})
	}

	// cases where setup is disabled through configs
	cases = map[string]Options{
		"DisableSetup": {
			EnableSetup: false,
		},
		"AdminAccessKey": {
			EnableSetup:    true,
			AdminAccessKey: "admin-access-key",
		},
		"AccessKey": {
			EnableSetup: true,
			AccessKey:   "access-key",
		},
		"ImportProviders": {
			EnableSetup: true,
			Import: &config.Config{
				Providers: []config.Provider{
					{
						Name: "provider",
					},
				},
			},
		},
		"ImportGrants": {
			EnableSetup: true,
			Import: &config.Config{
				Grants: []config.Grant{
					{
						Role: "admin",
					},
				},
			},
		},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			s.options = options
			require.False(t, s.setupRequired())
		})
	}

	// reset options
	s.options = Options{
		EnableSetup: true,
	}

	err := db.Create(&models.Machine{Name: "non-admin"}).Error
	require.NoError(t, err)

	require.True(t, s.setupRequired())

	err = db.Create(&models.Machine{Name: "admin"}).Error
	require.NoError(t, err)

	require.False(t, s.setupRequired())
}
