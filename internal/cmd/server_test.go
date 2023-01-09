package cmd

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"

	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/redis"
	"github.com/infrahq/infra/internal/testing/database"
)

func TestServerCmd_LoadOptions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows

	type testCase struct {
		name        string
		setup       func(t *testing.T, cmd *cobra.Command)
		expectedErr string
		expected    func(t *testing.T) server.Options
	}

	run := func(t *testing.T, tc testCase) {
		patchRunServer(t, noServerRun)
		var actual server.Options
		patchNewServer(t, &actual)

		cmd := newServerCmd()
		cmd.SetArgs([]string{}) // prevent reading of os.Args
		if tc.setup != nil {
			tc.setup(t, cmd)
		}

		err := cmd.Execute()
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr)
			return
		}

		assert.NilError(t, err)
		expected := tc.expected(t)
		assert.DeepEqual(t, expected, actual)
	}

	testCases := []testCase{
		{
			name: "config filename specified as env var",
			setup: func(t *testing.T, cmd *cobra.Command) {
				content := `
                    addr:
                      http: "127.0.0.1:1455"`

				dir := fs.NewDir(t, t.Name(),
					fs.WithFile("cfg.yaml", content))

				t.Setenv("INFRA_SERVER_CONFIG_FILE", dir.Join("cfg.yaml"))
			},
			expected: func(t *testing.T) server.Options {
				expected := defaultServerOptions(filepath.Join(dir, ".infra"))
				expected.Addr.HTTP = "127.0.0.1:1455"
				return expected
			},
		},
		{
			name: "env var can set a value outside of the top level",
			setup: func(t *testing.T, cmd *cobra.Command) {
				t.Setenv("INFRA_SERVER_ADDR_HTTP", "127.0.0.1:1455")
			},
			expected: func(t *testing.T) server.Options {
				expected := defaultServerOptions(filepath.Join(dir, ".infra"))
				expected.Addr.HTTP = "127.0.0.1:1455"
				return expected
			},
		},
		{
			name: "parse ui-proxy-url from command line flag",
			setup: func(t *testing.T, cmd *cobra.Command) {
				cmd.SetArgs([]string{"--ui-proxy-url", "https://127.0.1.2:34567"})
			},
			expected: func(t *testing.T) server.Options {
				expected := defaultServerOptions(filepath.Join(dir, ".infra"))
				expected.UI.ProxyURL = types.URL{
					Scheme: "https",
					Host:   "127.0.1.2:34567",
				}
				return expected
			},
		},
		{
			name: "parse ui-proxy-url from config file",
			setup: func(t *testing.T, cmd *cobra.Command) {
				content := `
                  ui:
                    proxyURL: https://127.0.1.2:34567
`
				dir := fs.NewDir(t, t.Name(),
					fs.WithFile("cfg.yaml", content))
				cmd.SetArgs([]string{"--config-file", dir.Join("cfg.yaml")})
			},
			expected: func(t *testing.T) server.Options {
				expected := defaultServerOptions(filepath.Join(dir, ".infra"))
				expected.UI.ProxyURL = types.URL{
					Scheme: "https",
					Host:   "127.0.1.2:34567",
				}
				return expected
			},
		},
		{
			name: "env vars with config file",
			setup: func(t *testing.T, cmd *cobra.Command) {
				content := `
grants:
  - user: user1
    resource: infra
    role: admin
  - user: user2
    resource: infra
    role: admin

users:
  - name: username
    accessKey: access-key
    password: the-password
`
				dir := fs.NewDir(t, t.Name(),
					fs.WithFile("cfg.yaml", content))
				cmd.SetArgs([]string{"--config-file", dir.Join("cfg.yaml")})
				t.Setenv("INFRA_SERVER_TLS_CA", "foo/ca.crt")
				t.Setenv("INFRA_SERVER_TLS_CA_PRIVATE_KEY", "file:foo/ca.key")
				t.Setenv("INFRA_SERVER_DB_CONNECTION_STRING", "host=db port=5432 user=postgres dbname=postgres password=postgres")
				t.Setenv("INFRA_SERVER_DB_ENCRYPTION_KEY", "/root.key")

			},
			expected: func(t *testing.T) server.Options {
				expected := defaultServerOptions(filepath.Join(dir, ".infra"))
				expected.TLS.CA = "foo/ca.crt"
				expected.TLS.CAPrivateKey = "file:foo/ca.key"
				expected.DBConnectionString = "host=db port=5432 user=postgres dbname=postgres password=postgres"
				expected.DBEncryptionKey = "/root.key"
				expected.Config.Users = []server.User{
					{Name: "username", AccessKey: "access-key", Password: "the-password"},
				}
				expected.Config.Grants = []server.Grant{
					{User: "user1", Resource: "infra", Role: "admin"},
					{User: "user2", Resource: "infra", Role: "admin"},
				}
				return expected
			},
		},
		{
			name: "all options from config",
			setup: func(t *testing.T, cmd *cobra.Command) {
				content := `
version: 0.3
tlsCache: /cache/dir
enableTelemetry: false # default is true
enableSignup: false    # default is true
enableLogSampling: false # default is true
sessionDuration: 3m
sessionInactivityTimeout: 1m

dbEncryptionKey: /this-is-the-path
dbHost: the-host
dbPort: 5432
dbName: infradbname
dbUsername: infra
dbPassword: env:POSTGRES_DB_PASSWORD
dbParameters: sslmode=require

baseDomain: foo.example.com
loginDomainPrefix: login
googleClientID: aaa
googleClientSecret: bbb

tls:
  ca: testdata/ca.crt
  caPrivateKey: file:ca.key
  certificate: testdata/server.crt
  privateKey: file:server.key
  ACME: true

keys:
  - kind: vault
    config:
      token: the-token
      address: 10.1.1.1:1234

secrets:
  - kind: env
    name: base64env
    config:
      base64: true

addr:
  http: "1.2.3.4:23"
  https: "1.2.3.5:433"
  metrics: "1.2.3.6:8888"

ui:
  enabled: false # default is true
  proxyURL: "1.2.3.4:5151"

providers:
  - name: okta
    url: https://dev-okta.com/
    clientID: client-id
    clientSecret: the-secret

grants:
  - user: user1
    resource: infra
    role: admin
  - group: group1
    resource: production
    role: special

users:
  - name: username
    accessKey: access-key
    password: the-password

redis:
  host: myredis
  username: myuser
  password: mypassword

api:
  requestTimeout: 2m
  blockingRequestTimeout: 4m

`

				dir := fs.NewDir(t, t.Name(),
					fs.WithFile("cfg.yaml", content))
				cmd.SetArgs([]string{"--config-file", dir.Join("cfg.yaml")})
			},
			expected: func(t *testing.T) server.Options {
				return server.Options{
					Version:                  0.3,
					TLSCache:                 "/cache/dir",
					SessionDuration:          3 * time.Minute,
					SessionInactivityTimeout: 1 * time.Minute,

					DBEncryptionKey: "/this-is-the-path",
					DBHost:          "the-host",
					DBPort:          5432,
					DBParameters:    "sslmode=require",
					DBPassword:      "env:POSTGRES_DB_PASSWORD",
					DBUsername:      "infra",
					DBName:          "infradbname",

					BaseDomain:         "foo.example.com",
					LoginDomainPrefix:  "login",
					GoogleClientID:     "aaa",
					GoogleClientSecret: "bbb",

					Addr: server.ListenerOptions{
						HTTP:    "1.2.3.4:23",
						HTTPS:   "1.2.3.5:433",
						Metrics: "1.2.3.6:8888",
					},

					UI: server.UIOptions{
						ProxyURL: types.URL(url.URL{
							Scheme: "http",
							Host:   "1.2.3.4:5151",
						}),
					},

					TLS: server.TLSOptions{
						CA:           "-----BEGIN CERTIFICATE-----\nnot a real ca certificate\n-----END CERTIFICATE-----\n",
						CAPrivateKey: "file:ca.key",
						Certificate:  "-----BEGIN CERTIFICATE-----\nnot a real server certificate\n-----END CERTIFICATE-----\n",
						PrivateKey:   "file:server.key",
						ACME:         true,
					},

					Keys: []server.KeyProvider{
						{
							Kind: "vault",
							Config: server.VaultConfig{
								Token:   "the-token",
								Address: "10.1.1.1:1234",
							},
						},
					},

					Secrets: []server.SecretProvider{
						{
							Kind:   "env",
							Name:   "base64env",
							Config: server.GenericConfig{Base64: true},
						},
					},

					Config: server.Config{
						Providers: []server.Provider{
							{
								Name:         "okta",
								URL:          "https://dev-okta.com/",
								ClientID:     "client-id",
								ClientSecret: "the-secret",
							},
						},
						Grants: []server.Grant{
							{
								User:     "user1",
								Resource: "infra",
								Role:     "admin",
							},
							{
								Group:    "group1",
								Resource: "production",
								Role:     "special",
							},
						},
						Users: []server.User{
							{
								Name:      "username",
								AccessKey: "access-key",
								Password:  "the-password",
							},
						},
					},

					DB: data.NewDBOptions{
						MaxOpenConnections: 100,
						MaxIdleConnections: 100,
						MaxIdleTimeout:     5 * time.Minute,
					},

					Redis: redis.Options{
						Host:     "myredis",
						Port:     6379,
						Username: "myuser",
						Password: "mypassword",
					},

					API: server.APIOptions{
						RequestTimeout:         2 * time.Minute,
						BlockingRequestTimeout: 4 * time.Minute,
					},
				}
			},
		},
		{
			name: "options from all flags",
			setup: func(t *testing.T, cmd *cobra.Command) {
				cmd.SetArgs([]string{
					"--db-name", "database-name",
					"--db-port", "12345",
					"--db-host", "thehostname",
					"--enable-telemetry=false",
					"--session-duration", "3m",
					"--session-inactivity-timeout", "1m",
					"--enable-signup=false",
				})
			},
			expected: func(t *testing.T) server.Options {
				expected := defaultServerOptions(filepath.Join(dir, ".infra"))
				expected.DBName = "database-name"
				expected.DBHost = "thehostname"
				expected.DBPort = 12345
				expected.EnableTelemetry = false
				expected.SessionDuration = 3 * time.Minute
				expected.SessionInactivityTimeout = 1 * time.Minute
				expected.EnableSignup = false
				expected.BaseDomain = ""
				return expected
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestServerCmd_WithSecretsConfig(t *testing.T) {
	pgDriver := database.PostgresDriver(t, "_cmd")
	patchRunServer(t, noServerRun)

	rootKeyPath := filepath.Join(t.TempDir(), "root.key")
	content := `
      dbConnectionString: ` + pgDriver.DSN + `
      dbEncryptionKey: ` + rootKeyPath + `
      addr:
        http: "127.0.0.1:0"
        https: "127.0.0.1:0"
        metrics: "127.0.0.1:0"

      tls:
        ca: testdata/pki/localhost.crt
        caPrivateKey: file:testdata/pki/localhost.key

      secrets:
        - kind: env
          name: base64env
          config:
            base64: true
      keys:
        - kind: native
          config:
            secretProvider: base64env
`

	dir := fs.NewDir(t, t.Name(), fs.WithFile("cfg.yaml", content))
	t.Setenv("HOME", dir.Path())

	ctx := context.Background()
	err := Run(ctx, "server", "--config-file", dir.Join("cfg.yaml"))
	assert.NilError(t, err)
}

func patchRunServer(t *testing.T, fn func(context.Context, *server.Server) error) {
	orig := runServer
	runServer = fn
	t.Cleanup(func() {
		runServer = orig
	})
}

func noServerRun(context.Context, *server.Server) error {
	return nil
}

func patchNewServer(t *testing.T, target *server.Options) {
	orig := newServer
	t.Cleanup(func() {
		newServer = orig
	})

	newServer = func(options server.Options) (*server.Server, error) {
		*target = options
		return &server.Server{}, nil
	}
}

func TestServerCmd_NoFlagDefaults(t *testing.T) {
	cmd := newServerCmd()
	flags := cmd.Flags()
	err := flags.Parse(nil)
	assert.NilError(t, err)

	msg := "The default value of flags on the 'infra server' command will be ignored. " +
		"Set a default value in defaultServerOptions instead."
	flags.VisitAll(func(flag *pflag.Flag) {
		if sv, ok := flag.Value.(pflag.SliceValue); ok {
			if len(sv.GetSlice()) > 0 {
				t.Fatalf("Flag --%v uses non-zero value %v. %v", flag.Name, flag.Value, msg)
			}
			return
		}

		v := reflect.Indirect(reflect.ValueOf(flag.Value))
		if !v.IsZero() {
			t.Fatalf("Flag --%v uses non-zero value %v. %v", flag.Name, flag.Value, msg)
		}
	})
}

func TestCanonicalPath(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	t.Setenv("USERPROFILE", "/home/user")
	wd, err := filepath.EvalSymlinks(t.TempDir())
	assert.NilError(t, err)

	env.ChangeWorkingDir(t, wd)

	type testCase struct {
		path     string
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := canonicalPath(tc.path)
		assert.NilError(t, err)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{path: "/already/abs", expected: "/already/abs"},
		{path: "relative/no/dot", expected: wd + "/relative/no/dot"},
		{path: "./relative/dot", expected: wd + "/relative/dot"},
		{path: "$HOME/dir", expected: "/home/user/dir"},
		{path: "${HOME}/dir", expected: "/home/user/dir"},
		{path: "/not/$HOMEFOO/dir", expected: "/not/dir"},
		{path: "$HOMEFOO/dir", expected: "/dir"},
		{path: "~/config", expected: "/home/user/config"},
		{path: "~user/config", expected: wd + "/~user/config"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("in=%v out=%v", tc.path, tc.expected), func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestServerCmd_DeprecatedConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows

	type testCase struct {
		name        string
		setup       func(t *testing.T, cmd *cobra.Command)
		expectedErr string
	}

	run := func(t *testing.T, tc testCase) {
		patchRunServer(t, noServerRun)

		cmd := newServerCmd()
		cmd.SetArgs([]string{}) // prevent reading of os.Args
		if tc.setup != nil {
			tc.setup(t, cmd)
		}

		err := cmd.Execute()
		assert.ErrorContains(t, err, tc.expectedErr)
	}

	testCases := []testCase{
		{
			name: "dbEncryptionKeyProvider",
			setup: func(t *testing.T, cmd *cobra.Command) {
				t.Setenv("INFRA_SERVER_DB_ENCRYPTION_KEY_PROVIDER", "vault")
			},
			expectedErr: "dbEncryptionKeyProvider is no longer supported",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
