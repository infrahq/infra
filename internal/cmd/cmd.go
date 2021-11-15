package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

func blue(s string) string {
	return termenv.String(s).Bold().Foreground(termenv.ColorProfile().Color("#0057FF")).String()
}

// errWithResponseContext appends the response message to a returned error
func errWithResponseContext(err error, res *http.Response) error {
	var apiErr api.Error
	if decodeErr := json.NewDecoder(res.Body).Decode(&apiErr); decodeErr != nil {
		// ignore this decoding error and return the original error
		return err
	}

	return fmt.Errorf("%w (Message: %s)", err, apiErr.Message)
}

func NewAPIContext(token string) context.Context {
	return context.WithValue(context.Background(), api.ContextAccessToken, token)
}

func NewAPIClient(host string, skipTLSVerify bool) (*api.APIClient, error) {
	u, err := urlx.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("parsing host: %w", err)
	}

	config := api.NewConfiguration()
	config.Host = u.Host
	config.Scheme = "https"

	if skipTLSVerify {
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					//nolint:gosec // We may purposely set insecureskipverify via a flag
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return api.NewAPIClient(config), nil
}

func infraHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	infraDir := filepath.Join(homeDir, ".infra")

	err = os.MkdirAll(infraDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	return infraDir, nil
}

func apiContextFromConfig(host string) (context.Context, error) {
	config, err := readHostConfig(host)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, ErrConfigNotFound
	}

	return NewAPIContext(config.Token), nil
}

func apiClientFromConfig(host string) (*api.APIClient, error) {
	config, err := readHostConfig(host)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, ErrConfigNotFound
	}

	return NewAPIClient(config.Host, config.SkipTLSVerify)
}

func newLoginCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "login [HOST]",
		Short:   "Login to Infra",
		Args:    cobra.MaximumNArgs(1),
		Example: "$ infra login infra.example.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options LoginOptions
			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			if len(args) == 1 {
				options.Host = args[0]
			}

			return login(&options)
		},
	}

	cmd.Flags().DurationP("timeout", "t", defaultTimeout, "login timeout")

	return cmd, nil
}

func newLogoutCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "Logout Infra",
		Args:    cobra.MaximumNArgs(1),
		Example: "$ infra logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options LogoutOptions
			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			if len(args) == 1 {
				options.Host = args[0]
			}

			return logout(&options)
		},
	}

	return cmd, nil
}

func newListCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List destinations",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options ListOptions
			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			return list(&options)
		},
	}

	return cmd, nil
}

func newStartCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:    "start",
		Short:  "Start Infra",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var options registry.Options
			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			return registry.Run(options)
		},
	}

	cmd.Flags().StringP("config-path", "c", "", "Infra config file")
	cmd.Flags().String("root-api-key", "", "root API key")
	cmd.Flags().String("engine-api-key", "", "engine registration API key")
	cmd.Flags().String("tls-cache", "", "path to cache self-signed and Let's Encrypt TLS certificates")
	cmd.Flags().String("db-file", "", "path to database file")
	cmd.Flags().String("pg.host", "", "PostgreSQL host")
	cmd.Flags().Int("pg.port", -1, "PostgreSQL port")
	cmd.Flags().String("pg.dbname", "", "PostgreSQL database name")
	cmd.Flags().String("pg.user", "", "PostgreSQL user")
	cmd.Flags().String("pg.password", "", "PostgreSQL password")
	cmd.Flags().String("pg.parameters", "", "PostgreSQL connection parameters")
	cmd.Flags().String("pg.url", "", "PostgreSQL connection URL, overrides other postgres inputs")

	cmd.Flags().String("ui-proxy", "", "proxy UI requests to this host")
	cmd.Flags().Bool("enable-ui", false, "enable UI")

	cmd.Flags().Duration("providers-sync-interval", registry.DefaultProvidersSyncInterval, "the interval at which Infra will poll identity providers for users and groups")
	cmd.Flags().Duration("destinations-sync-interval", registry.DefaultDestinationsSyncInterval, "the interval at which Infra will poll destinations")

	cmd.Flags().Bool("enable-telemetry", true, "enable telemetry")
	cmd.Flags().Bool("enable-crash-reporting", true, "enable crash reporting")

	cmd.Flags().DurationP("session-duration", "d", registry.DefaultSessionDuration, "session duration")

	return cmd, nil
}

func newEngineCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:    "engine",
		Short:  "Start Infra Engine",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var options engine.Options
			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			return engine.Run(&options)
		},
	}

	cmd.Flags().StringP("name", "n", "", "destination name")
	cmd.Flags().StringP("kind", "k", "", "destination kind")
	cmd.Flags().String("api-key", "", "engine registry API key")
	cmd.Flags().String("tls-cache", "", "path to cache self-signed and Let's Encrypt TLS certificates")
	cmd.Flags().Bool("skip-tls-verify", true, "skip TLS verification")

	return cmd, nil
}

func newVersionCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display the Infra build version",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options VersionOptions
			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			return version(&options)
		},
	}

	cmd.Flags().Bool("client", false, "Display client version only")
	cmd.Flags().Bool("server", false, "Display server version only")

	return cmd, nil
}

func newTokensCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Token subcommands",
	}

	tokenCreateCmd, err := newTokenCreateCmd()
	if err != nil {
		return nil, err
	}

	cmd.AddCommand(tokenCreateCmd)

	return cmd, nil
}

func newKubernetesCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "kubernetes",
		Short:   "Kubernetes subcommands",
		Aliases: []string{"k", "k8s"},
	}

	kubernetesUseCmd, err := newKubernetesUseCmd()
	if err != nil {
		return nil, err
	}

	cmd.AddCommand(kubernetesUseCmd)

	return cmd, nil
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	loginCmd, err := newLoginCmd()
	if err != nil {
		return nil, err
	}

	logoutCmd, err := newLogoutCmd()
	if err != nil {
		return nil, err
	}

	listCmd, err := newListCmd()
	if err != nil {
		return nil, err
	}

	tokensCmd, err := newTokensCmd()
	if err != nil {
		return nil, err
	}

	kubernetesCmd, err := newKubernetesCmd()
	if err != nil {
		return nil, err
	}

	versionCmd, err := newVersionCmd()
	if err != nil {
		return nil, err
	}

	startCmd, err := newStartCmd()
	if err != nil {
		return nil, err
	}

	engineCmd, err := newEngineCmd()
	if err != nil {
		return nil, err
	}

	rootCmd := &cobra.Command{
		Use:   "infra",
		Short: "Infrastructure Identity & Access Management (IAM)",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			var options internal.Options
			if err := internal.ParseOptions(cmd, &options); err != nil {
				return err
			}

			logger, err := logging.Initialize(options.V)
			if err != nil {
				logging.L.Warn(err.Error())
			} else {
				logging.L = logger
				logging.S = logger.Sugar()
			}

			return nil
		},
	}

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(tokensCmd)
	rootCmd.AddCommand(kubernetesCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(engineCmd)

	rootCmd.PersistentFlags().StringP("config-file", "f", "", "Infra configuration file path")
	rootCmd.PersistentFlags().StringP("host", "H", "", "Infra host")
	rootCmd.PersistentFlags().CountP("v", "v", "Log verbosity")

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
