package cmd

import (
	"context"
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
		t.Setenv("HOME", "/home/user")
		t.Setenv("USERPROFILE", "/home/user") // Windows

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
			name: "secret providers",
			setup: func(t *testing.T, cmd *cobra.Command) {
				content := `
                    secrets:
                      - kind: env
                        name: base64env
                        config:
                          base64: true`

				dir := fs.NewDir(t, t.Name(),
					fs.WithFile("cfg.yaml", content))
				cmd.SetArgs([]string{"--config-file", dir.Join("cfg.yaml")})
			},
			expected: func(t *testing.T) server.Options {
				expected := serverOptionsWithDefaults()
				expected.Secrets = []server.SecretProvider{
					{
						Kind:   "env",
						Name:   "base64env",
						Config: server.GenericConfig{Base64: true},
					},
				}
				return expected
			},
		},
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
				expected := serverOptionsWithDefaults()
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
				expected := serverOptionsWithDefaults()
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
				expected := serverOptionsWithDefaults()
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
				expected := serverOptionsWithDefaults()
				expected.UI.ProxyURL = types.URL{
					Scheme: "https",
					Host:   "127.0.1.2:34567",
				}
				expected.UI.Enabled = true
				return expected
			},
		},
		{
			name: "options from all flags",
			setup: func(t *testing.T, cmd *cobra.Command) {
				cmd.SetArgs([]string{
					"--db-name", "database-name",
					"--db-file", "$HOME/database-filename",
					"--db-port", "12345",
					"--db-host", "thehostname",
					"--enable-telemetry=false",
					"--session-duration", "3m",
					"--enable-signup=false",
				})
			},
			expected: func(t *testing.T) server.Options {
				expected := serverOptionsWithDefaults()
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

// serverOptionsWithDefaults returns all the default values. Some default values
// include placeholders for environment variables that will be resolved by the
// command.
// TODO: resolve the path in defaultServerOptions instead, and remove this function
func serverOptionsWithDefaults() server.Options {
	o := defaultServerOptions()
	o.TLSCache = "/home/user/.infra/cache"
	o.DBFile = "/home/user/.infra/sqlite3.db"
	o.DBEncryptionKey = "/home/user/.infra/sqlite3.db.key"
	return o
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
