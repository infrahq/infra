package cmd

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/goware/urlx"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry"
)

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

func defaultAPIClient() (*api.Client, error) {
	config, err := readHostConfig("")
	if err != nil {
		return nil, err
	}

	return apiClient(config.Host, config.Token, config.SkipTLSVerify)
}

func apiClient(host string, token string, skipTLSVerify bool) (*api.Client, error) {
	u, err := urlx.Parse(host)
	if err != nil {
		return nil, err
	}

	u.Scheme = "https"

	return &api.Client{
		Url:   u.String(),
		Token: token,
		Http: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					//nolint:gosec // We may purposely set insecureskipverify via a flag
					InsecureSkipVerify: skipTLSVerify,
				},
			},
		},
	}, nil
}

func newLoginCmd() *cobra.Command {
	var options LoginOptions

	cmd := &cobra.Command{
		Use:     "login [SERVER]",
		Short:   "Login to Infra",
		Example: "$ infra login",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				options.Host = args[0]
			}

			if err := login(&options); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().DurationVarP(&options.Timeout, "timeout", "t", defaultTimeout, "login timeout")

	return cmd
}

var logoutCmd = &cobra.Command{
	Use:     "logout",
	Short:   "Logout of Infra",
	Example: "$ infra logout",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := logout(); err != nil {
			return err
		}

		return nil
	},
}

func listCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List infrastructure",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list()
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "list all infrastructure (default shows infrastructure you have access to)")

	return cmd
}

func newServerCmd() (*cobra.Command, error) {
	var (
		options    registry.Options
		configFile string
	)

	var err error

	parseConfig := func() {
		if configFile == "" {
			return
		}

		var contents []byte

		contents, err = ioutil.ReadFile(configFile)
		if err != nil {
			return
		}

		err = yaml.Unmarshal(contents, &options)
	}

	cobra.OnInitialize(parseConfig)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start Infra Server",

		RunE: func(cmd *cobra.Command, args []string) error {
			if err != nil {
				return err
			}

			return registry.Run(options)
		},
	}

	infraDir, err := infraHomeDir()
	if err != nil {
		return nil, err
	}

	cmd.Flags().StringVarP(&configFile, "config-file", "f", "", "Server configuration file")
	cmd.Flags().StringVar(&options.RootAPIToken, "root-api-token", "file:"+filepath.Join(infraDir, "root-api-token"), "Root API token (secret)")
	cmd.Flags().StringVar(&options.EngineAPIToken, "engine-api-token", "file:"+filepath.Join(infraDir, "engine-api-token"), "Engine API token (secret)")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(infraDir, "tls"), "Directory to cache TLS certificates")
	cmd.Flags().StringVar(&options.DBFile, "db-file", filepath.Join(infraDir, "db"), "Path to database file")
	cmd.Flags().StringVar(&options.DBEncryptionKey, "db-encryption-key", filepath.Join(infraDir, "key"), "Database encryption key")
	cmd.Flags().StringVar(&options.DBEncryptionKeyProvider, "db-encryption-key-provider", "native", "Database encryption key provider")
	cmd.Flags().StringVar(&options.DBHost, "db-host", "", "Database host")
	cmd.Flags().IntVar(&options.DBPort, "db-port", 5432, "Database port")
	cmd.Flags().StringVar(&options.DBName, "db-name", "", "Database name")
	cmd.Flags().StringVar(&options.DBUser, "db-user", "", "Database user")
	cmd.Flags().StringVar(&options.DBPassword, "db-password", "", "Database password (secret)")
	cmd.Flags().StringVar(&options.DBParameters, "db-parameters", "", "Database additional connection parameters")
	cmd.Flags().BoolVar(&options.EnableTelemetry, "enable-telemetry", true, "Enable telemetry")
	cmd.Flags().BoolVar(&options.EnableCrashReporting, "enable-crash-reporting", true, "Enable crash reporting")
	cmd.Flags().DurationVarP(&options.SessionDuration, "session-duration", "d", registry.DefaultSessionDuration, "Session duration")

	return cmd, nil
}

func newEngineCmd() *cobra.Command {
	var (
		options          engine.Options
		engineConfigFile string
		err              error
	)

	parseConfig := func() {
		if engineConfigFile == "" {
			return
		}

		var contents []byte

		contents, err = ioutil.ReadFile(engineConfigFile)
		if err != nil {
			return
		}

		err = yaml.Unmarshal(contents, &options)
	}

	cobra.OnInitialize(parseConfig)

	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Start Infra Engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err != nil {
				return err
			}

			if err := engine.Run(options); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&engineConfigFile, "config-file", "f", "", "Engine config file")
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "Destination name")
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "kubernetes", "Destination kind")
	cmd.Flags().StringVar(&options.APIToken, "api-token", "", "Engine API token (use file:// to load from a file)")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", "", "Path to cache self-signed and Let's Encrypt TLS certificates")
	cmd.Flags().StringVar(&options.Server, "server", "", "Infra Server hostname")
	cmd.Flags().BoolVar(&options.SkipTLSVerify, "skip-tls-verify", true, "Skip TLS verification")

	return cmd
}

func newUseCmd() *cobra.Command {
	var (
		namespace string
		labels    []string
	)

	cmd := &cobra.Command{
		Use:   "use [INFRASTRUCTURE]",
		Short: "Connect to infrastructure",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			if err := use(&UseOptions{
				Name:      name,
				Namespace: namespace,
				Labels:    labels,
			}); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace")
	cmd.Flags().StringSliceVarP(&labels, "labels", "l", []string{}, "Labels")

	return cmd
}

var tokensCreateCmd = &cobra.Command{
	Use:   "create DESTINATION",
	Short: "Create a JWT token for connecting to a destination, e.g. Kubernetes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := tokensCreate(args[0]); err != nil {
			return err
		}

		return nil
	},
}

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Create & manage identity tokens",
	}

	cmd.AddCommand(tokensCreateCmd)

	return cmd
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the Infra version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return version()
	},
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	var level string

	rootCmd := &cobra.Command{
		Use:               "infra",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.SetLevel(level)
		},
	}

	serverCmd, err := newServerCmd()
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(newTokensCmd())
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(newEngineCmd())
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringVar(&level, "log-level", "info", "Log level (error, warn, info, debug)")

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
