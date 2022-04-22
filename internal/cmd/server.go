package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server"
)

func newServerCmd() *cobra.Command {
	options := defaultServerOptions()
	var configFilename string

	cmd := &cobra.Command{
		Use:    "server",
		Short:  "Start Infra server",
		Args:   NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logging.SetServerLogger()

			if configFilename == "" {
				configFilename = os.Getenv("INFRA_SERVER_CONFIG_FILE")
			}

			err := cliopts.Load(&options, cliopts.Options{
				Filename:  configFilename,
				EnvPrefix: "INFRA_SERVER",
				Flags:     cmd.Flags(),
			})
			if err != nil {
				return err
			}

			tlsCache, err := canonicalPath(options.TLSCache)
			if err != nil {
				return err
			}

			options.TLSCache = tlsCache

			dbFile, err := canonicalPath(options.DBFile)
			if err != nil {
				return err
			}

			options.DBFile = dbFile

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
	cmd.Flags().String("tls-cache", "$HOME/.infra/cache", "Directory to cache TLS certificates")
	cmd.Flags().String("db-file", "$HOME/.infra/sqlite3.db", "Path to SQLite 3 database")
	cmd.Flags().String("db-name", "", "Database name")
	cmd.Flags().String("db-host", "", "Database host")
	cmd.Flags().Int("db-port", 0, "Database port")
	cmd.Flags().String("db-username", "", "Database username")
	cmd.Flags().String("db-password", "", "Database password (secret)")
	cmd.Flags().String("db-parameters", "", "Database additional connection parameters")
	cmd.Flags().String("db-encryption-key", "$HOME/.infra/sqlite3.db.key", "Database encryption key")
	cmd.Flags().String("db-encryption-key-provider", "native", "Database encryption key provider")
	cmd.Flags().Bool("enable-telemetry", true, "Enable telemetry")
	cmd.Flags().Bool("enable-crash-reporting", true, "Enable crash reporting")
	cmd.Flags().BoolVar(&options.UI.Enabled, "enable-ui", false, "Enable Infra server UI")
	cmd.Flags().Var(&options.UI.ProxyURL, "ui-proxy-url", "Proxy upstream UI requests to this url")
	cmd.Flags().Duration("session-duration", time.Hour*12, "User session duration")
	cmd.Flags().Bool("enable-signup", true, "Enable one-time admin signup")

	return cmd
}

func defaultServerOptions() server.Options {
	return server.Options{
		Addr: server.ListenerOptions{
			HTTP:    ":80",
			HTTPS:   ":443",
			Metrics: ":9090",
		},
	}
}

// runServer is a shim for testing.
var runServer = func(ctx context.Context, srv *server.Server) error {
	return srv.Run(ctx)
}

// newServer is a shim for testing.
var newServer = server.New
