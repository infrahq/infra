package cmd

import (
	"context"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/server"
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
                    enabled: true
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
				expected.UI.Enabled = true
				return expected
			},
		},
		{
			name: "all options from config",
			setup: func(t *testing.T, cmd *cobra.Command) {
				content := `
version: 0.2
tlsCache: /cache/dir
enableTelemetry: false # default is true
enableSignup: false    # default is true
sessionDuration: 3m

dbFile: /db/file
dbEncryptionKey: /this-is-the-path
dbEncryptionKeyProvider: the-provider
dbHost: the-host
dbPort: 5432
dbName: infradbname
dbUsername: infra
dbPassword: env:POSTGRES_DB_PASSWORD
dbParameters: sslmode=require

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

`

				dir := fs.NewDir(t, t.Name(),
					fs.WithFile("cfg.yaml", content))
				cmd.SetArgs([]string{"--config-file", dir.Join("cfg.yaml")})
			},
			expected: func(t *testing.T) server.Options {
				return server.Options{
					Version:         0.2,
					TLSCache:        "/cache/dir",
					SessionDuration: 3 * time.Minute,

					DBEncryptionKey:         "/this-is-the-path",
					DBEncryptionKeyProvider: "the-provider",
					DBFile:                  "/db/file",
					DBHost:                  "the-host",
					DBPort:                  5432,
					DBParameters:            "sslmode=require",
					DBPassword:              "env:POSTGRES_DB_PASSWORD",
					DBUsername:              "infra",
					DBName:                  "infradbname",

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
				}
			},
		},
		{
			name: "options from all flags",
			setup: func(t *testing.T, cmd *cobra.Command) {
				cmd.SetArgs([]string{
					"--db-name", "database-name",
					"--db-file", "/home/user/database-filename",
					"--db-port", "12345",
					"--db-host", "thehostname",
					"--enable-telemetry=false",
					"--session-duration", "3m",
					"--enable-signup=false",
				})
			},
			expected: func(t *testing.T) server.Options {
				expected := defaultServerOptions(filepath.Join(dir, ".infra"))
				expected.DBName = "database-name"
				expected.DBFile = "/home/user/database-filename"
				expected.DBHost = "thehostname"
				expected.DBPort = 12345
				expected.EnableTelemetry = false
				expected.SessionDuration = 3 * time.Minute
				expected.EnableSignup = false
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
	patchRunServer(t, noServerRun)

	content := `
      addr:
        http: "127.0.0.1:0"
        https: "127.0.0.1:0"
        metrics: "127.0.0.1:0"
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
