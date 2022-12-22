package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/redis"
)

func newServerCmd() *cobra.Command {
	var configFilename string

	cmd := &cobra.Command{
		Use:     "server",
		Short:   "Start the Infra server",
		Args:    NoArgs,
		GroupID: groupServices,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logging.UseServerLogger()

			if configFilename == "" {
				configFilename = os.Getenv("INFRA_SERVER_CONFIG_FILE")
			}

			infraDir, err := infraHomeDir()
			if err != nil {
				return err
			}
			options := defaultServerOptions(infraDir)

			if err := server.ApplyOptions(&options, configFilename, cmd.Flags()); err != nil {
				return err
			}

			tlsCache, err := canonicalPath(options.TLSCache)
			if err != nil {
				return err
			}

			options.TLSCache = tlsCache

			dbEncryptionKey, err := canonicalPath(options.DBEncryptionKey)
			if err != nil {
				return err
			}

			options.DBEncryptionKey = dbEncryptionKey

			srv, err := newServer(options)
			if err != nil {
				return fmt.Errorf("creating server: %w", err)
			}
			return runServer(cmd.Context(), srv)
		},
	}

	cmd.Flags().StringVarP(&configFilename, "config-file", "f", "", "Server configuration file")
	cmd.Flags().String("tls-cache", "", "Directory to cache TLS certificates")
	cmd.Flags().String("db-name", "", "Database name")
	cmd.Flags().String("db-host", "", "Database host")
	cmd.Flags().Int("db-port", 0, "Database port")
	cmd.Flags().String("db-username", "", "Database username")
	cmd.Flags().String("db-password", "", "Database password (secret)")
	cmd.Flags().String("db-parameters", "", "Database additional connection parameters")
	cmd.Flags().String("db-encryption-key", "", "Database encryption key")
	cmd.Flags().String("db-encryption-key-provider", "", "Database encryption key provider")
	cmd.Flags().Bool("enable-telemetry", false, "Enable telemetry")
	cmd.Flags().Var(&types.URL{}, "ui-proxy-url", "Enable UI and proxy requests to this url")
	cmd.Flags().Duration("session-duration", 0, "Maximum session duration per user login")
	cmd.Flags().Duration("session-inactivity-timeout", 0, "A user must interact with Infra at least once within this amount of time for their session to remain valid")
	cmd.Flags().Bool("enable-signup", false, "Enable one-time admin signup")
	cmd.Flags().String("base-domain", "", "base-domain for the server, eg example.com")
	cmd.Flags().String("login-domain-prefix", "", "The path prefix on the base-domain that clients are redirected to after social login")
	cmd.Flags().String("google-client-id", "", "Client ID of the Google client used for social login")
	cmd.Flags().String("google-client-secret", "", "Client secret of the Google client used for social login")

	return cmd
}

func defaultServerOptions(infraDir string) server.Options {
	return server.Options{
		Version:                  0.3, // update this as the config version changes
		TLSCache:                 filepath.Join(infraDir, "cache"),
		DBEncryptionKey:          filepath.Join(infraDir, "sqlite3.db.key"),
		DBEncryptionKeyProvider:  "native",
		EnableTelemetry:          true,
		SessionDuration:          24 * time.Hour * 30, // 30 days
		SessionInactivityTimeout: 24 * time.Hour * 3,  // 3 days
		EnableSignup:             false,
		BaseDomain:               "",
		EnableLogSampling:        true,

		Addr: server.ListenerOptions{
			HTTP:    ":80",
			HTTPS:   ":443",
			Metrics: ":9090",
		},

		DB: data.NewDBOptions{
			MaxOpenConnections: 100,
			MaxIdleConnections: 100,
			MaxIdleTimeout:     5 * time.Minute,
		},

		Redis: redis.Options{
			Port: 6379,
		},

		API: server.APIOptions{
			RequestTimeout:         time.Minute,
			BlockingRequestTimeout: 5 * time.Minute,
		},
	}
}

// runServer is a shim for testing.
var runServer = func(ctx context.Context, srv *server.Server) error {
	return srv.Run(ctx)
}

// newServer is a shim for testing.
var newServer = server.New

func canonicalPath(path string) (string, error) {
	path = os.ExpandEnv(path)

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = strings.Replace(path, "~", homeDir, 1)
	}

	return filepath.Abs(path)
}
