package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/connector"
	"github.com/infrahq/infra/internal/logging"
)

func newConnectorCmd() *cobra.Command {
	var configFilename string

	cmd := &cobra.Command{
		Use:    "connector",
		Short:  "Start the Infra connector",
		Args:   NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logging.UseServerLogger()

			options := defaultConnectorOptions()
			err := cliopts.Load(&options, cliopts.Options{
				Filename:  configFilename,
				EnvPrefix: "INFRA_CONNECTOR",
				Flags:     cmd.Flags(),
			})
			if err != nil {
				return err
			}

			// backwards compat for old access key values with a prefix
			accessKey := options.Server.AccessKey.String()
			switch {
			case strings.HasPrefix(accessKey, "file:"):
				filename := strings.TrimPrefix(accessKey, "file:")
				if err := options.Server.AccessKey.Set(filename); err != nil {
					return err
				}
				logging.L.Warn().Msg("accessKey with 'file:' prefix is deprecated. Use the filename without the file: prefix instead.")
			case strings.HasPrefix(accessKey, "env:"):
				key := strings.TrimPrefix(accessKey, "env:")
				options.Server.AccessKey = types.StringOrFile(os.Getenv(key))
				logging.L.Warn().Msg("accessKey with 'env:' prefix is deprecated. Use the INFRA_ACCESS_KEY env var instead.")
			case strings.HasPrefix(accessKey, "plaintext:"):
				options.Server.AccessKey = types.StringOrFile(strings.TrimPrefix(accessKey, "plaintext:"))
				logging.L.Warn().Msg("accessKey with 'plaintext:' prefix is deprecated. Use the literal value without a prefix.")
			}

			// Also accept the same env var as the CLI for setting the access key
			if accessKey, ok := os.LookupEnv("INFRA_ACCESS_KEY"); ok {
				if err := options.Server.AccessKey.Set(accessKey); err != nil {
					return err
				}
			}
			return runConnector(cmd.Context(), options)
		},
	}

	cmd.Flags().StringVarP(&configFilename, "config-file", "f", "", "Connector config file")
	cmd.Flags().StringP("server-url", "s", "", "Infra server hostname")
	cmd.Flags().StringP("server-access-key", "a", "", "Infra access key (use file:// to load from a file)")
	cmd.Flags().StringP("name", "n", "", "Destination name")
	cmd.Flags().String("ca-cert", "", "Path to CA certificate file")
	cmd.Flags().String("ca-key", "", "Path to CA key file")
	cmd.Flags().Bool("server-skip-tls-verify", false, "Skip verifying server TLS certificates")

	return cmd
}

// runConnector is a shim for testing
var runConnector = connector.Run

func defaultConnectorOptions() connector.Options {
	return connector.Options{
		Addr: connector.ListenerOptions{
			HTTP:    ":80",
			HTTPS:   ":443",
			Metrics: ":9090",
		},
	}
}
