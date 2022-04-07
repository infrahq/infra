package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "server",
		Short:  "Start Infra server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logging.SetServerLogger()

			// override default strcase.ToLowerCamel behaviour
			strcase.ConfigureAcronym("enable-ui", "enableUI")
			strcase.ConfigureAcronym("ui-proxy-url", "uiProxyURL")

			options := defaultServerOptions()
			if err := parseOptions(cmd, &options, "INFRA_SERVER"); err != nil {
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

			srv, err := server.New(options)
			if err != nil {
				return fmt.Errorf("creating server: %w", err)
			}
			return runServer(cmd.Context(), srv)
		},
	}

	cmd.Flags().StringP("config-file", "f", "", "Server configuration file")
	cmd.Flags().String("admin-access-key", "", "Admin access key (secret)")
	cmd.Flags().String("access-key", "", "Access key (secret)")
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
	cmd.Flags().Bool("enable-ui", false, "Enable Infra server UI")
	cmd.Flags().String("ui-proxy-url", "", "Proxy upstream UI requests to this url")
	cmd.Flags().Duration("session-duration", time.Hour*12, "User session duration")
	cmd.Flags().Bool("enable-setup", true, "Enable one-time setup")

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

// shim for testing
var runServer = func(ctx context.Context, srv *server.Server) error {
	return srv.Run(ctx)
}
