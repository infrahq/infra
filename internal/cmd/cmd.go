package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/goware/urlx"
	"github.com/lensesio/tableprinter"
	"github.com/mcuadros/go-defaults"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/config"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server"
)

func parseOptions(cmd *cobra.Command, options interface{}, envPrefix string) error {
	v := viper.New()

	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	v.SetConfigName("infra")
	v.SetConfigType("yaml")

	v.AddConfigPath("/etc/infrahq")
	v.AddConfigPath("$HOME/.infra")
	v.AddConfigPath(".")

	if configFileFlag := cmd.Flags().Lookup("config-file"); configFileFlag != nil {
		if configFile := configFileFlag.Value.String(); configFile != "" {
			v.SetConfigFile(configFile)
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// workaround for viper not correctly binding env vars
	// https://github.com/spf13/viper/issues/761
	envKeys := make(map[string]interface{})
	if err := mapstructure.Decode(options, &envKeys); err != nil {
		return err
	}

	for envKey := range envKeys {
		if err := v.BindEnv(envKey); err != nil {
			return err
		}
	}

	defaults.SetDefaults(options)

	if err := v.ReadInConfig(); err != nil {
		var errNotFound *viper.ConfigFileNotFoundError
		if errors.As(err, &errNotFound) {
			return err
		}
	}

	return v.Unmarshal(options)
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

func printTable(data interface{}) {
	table := tableprinter.New(os.Stdout)

	table.HeaderAlignment = tableprinter.AlignLeft
	table.AutoWrapText = false
	table.DefaultAlignment = tableprinter.AlignLeft
	table.CenterSeparator = ""
	table.ColumnSeparator = ""
	table.RowSeparator = ""
	table.HeaderLine = false
	table.BorderBottom = false
	table.BorderLeft = false
	table.BorderRight = false
	table.BorderTop = false
	table.Print(data)
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
		Url:   fmt.Sprintf("%s://%s", u.Scheme, u.Host),
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
	type loginOptions struct {
		Host string `mapstructure:"host"`
	}

	return &cobra.Command{
		Use:     "login [SERVER]",
		Short:   "Login to Infra Server",
		Example: "$ infra login",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options loginOptions
			if err := parseOptions(cmd, &options, "INFRA"); err != nil {
				return err
			}

			if len(args) == 1 {
				options.Host = args[0]
			}

			return login(options.Host)
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "logout",
		Short:   "Logout of Infra",
		Example: "$ infra logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logout()
		},
	}
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List destinations and your access",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list()
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [DESTINATION]",
		Short: "Connect to a destination",
		Example: `
# Connect to a Kubernetes cluster
infra use kubernetes.development

# Connect to a Kubernetes namespace
infra use kubernetes.development.kube-system
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			config, err := currentHostConfig()
			if err != nil {
				return err
			}

			err = updateKubeconfig(client, config.ID)
			if err != nil {
				return err
			}

			parts := strings.Split(name, ".")

			if len(parts) < 2 {
				return errors.New("invalid argument")
			}

			if len(parts) <= 2 || parts[2] == "default" {
				return kubernetesSetContext("infra:" + parts[1])
			}

			return kubernetesSetContext("infra:" + parts[1] + ":" + parts[2])
		},
	}
}

func newAccessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access",
		Short: "Manage access",
	}

	cmd.AddCommand(newAccessListCmd())
	cmd.AddCommand(newAccessGrantCmd())
	cmd.AddCommand(newAccessRevokeCmd())

	return cmd
}

func newDestinationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destinations",
		Short: "Connect & manage destinations",
	}

	cmd.AddCommand(newDestinationsListCmd())
	cmd.AddCommand(newDestinationsAddCmd())
	cmd.AddCommand(newDestinationsRemoveCmd())

	return cmd
}

func newServerCmd() (*cobra.Command, error) {
	var (
		options    server.Options
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

			return server.Run(options)
		},
	}

	infraDir, err := infraHomeDir()
	if err != nil {
		return nil, err
	}

	cmd.Flags().StringVarP(&configFile, "config-file", "f", "", "Server configuration file")
	cmd.Flags().StringVar(&options.AdminAccessKey, "admin-access-key", "file:"+filepath.Join(infraDir, "admin-access-key"), "Admin access key (secret)")
	cmd.Flags().StringVar(&options.AccessKey, "access-key", "file:"+filepath.Join(infraDir, "access-key"), "Access key (secret)")
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
	cmd.Flags().BoolVar(&options.EnableUI, "enable-ui", false, "Enable ui")
	cmd.Flags().DurationVarP(&options.SessionDuration, "session-duration", "d", time.Hour*12, "Session duration")
	cmd.Flags().StringVar(&options.UIProxyURL, "ui-proxy", "", "Proxy upstream UI requests to this url")

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

	infraDir, err := infraHomeDir()
	if err != nil {
		return nil
	}

	cmd.Flags().StringVarP(&engineConfigFile, "config-file", "f", "", "Engine config file")
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "Destination name")
	cmd.Flags().StringVar(&options.AccessKey, "access-key", "", "Infra access key (use file:// to load from a file)")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(infraDir, "tls"), "Directory to cache TLS certificates")
	cmd.Flags().StringVar(&options.Server, "server", "", "Infra Server hostname")
	cmd.Flags().BoolVar(&options.SkipTLSVerify, "skip-tls-verify", true, "Skip TLS verification")

	return cmd
}

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Create & manage tokens",
	}

	cmd.AddCommand(newTokensCreateCmd())

	return cmd
}

func newProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Connect & manage identity providers",
	}

	cmd.AddCommand(newProvidersListCmd())
	cmd.AddCommand(newProvidersAddCmd())
	cmd.AddCommand(newProvidersRemoveCmd())

	return cmd
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Display the info about the current session",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := currentHostConfig()
			if err != nil {
				return err
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
			defer w.Flush()

			fmt.Fprintln(w)
			fmt.Fprintln(w, "Server:\t", config.Host)

			if config.ID == 0 {
				fmt.Fprintln(w, "User:\t", "system")
				fmt.Fprintln(w)
				return nil
			}

			provider, err := client.GetProvider(config.ProviderID)
			if err != nil {
				return err
			}

			fmt.Fprintln(w, "Identity Provider:\t", provider.Name, fmt.Sprintf("(%s)", provider.URL))

			user, err := client.GetUser(config.ID)
			if err != nil {
				return err
			}

			fmt.Fprintln(w, "User:\t", user.Email)

			groups, err := client.ListUserGroups(config.ID)
			if err != nil {
				return err
			}

			var names string
			for i, g := range groups {
				if i != 0 {
					names += ", "
				}

				names += g.Name
			}

			fmt.Fprintln(w, "Groups:\t", names)

			admin := false
			for _, p := range user.Permissions {
				if p == "infra.*" {
					admin = true
				}
			}

			fmt.Fprintln(w, "Admin:\t", admin)
			fmt.Fprintln(w)

			return nil
		},
	}
}

func newImportCmd() *cobra.Command {
	var replace bool

	cmd := &cobra.Command{
		Use:   "import [FILE]",
		Short: "Import an infra server configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			contents, err := ioutil.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("reading configuration file: %w", err)
			}

			var c config.Config
			err = yaml.Unmarshal(contents, &c)
			if err != nil {
				return err
			}

			return config.Import(client, c, replace)
		},
	}

	cmd.Flags().BoolVar(&replace, "replace", false, "replace any existing configuration")

	return cmd
}

func newMachinesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "machines",
		Short: "Create & manage machine identities",
	}

	cmd.AddCommand(newMachinesCreateCmd())
	cmd.AddCommand(newMachinesListCmd())
	cmd.AddCommand(newMachinesDeleteCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the Infra version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return version()
		},
	}
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	type rootOptions struct {
		LogLevel string `mapstructure:"log-level"`
	}

	rootCmd := &cobra.Command{
		Use:               "infra",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var options rootOptions
			if err := parseOptions(cmd, &options, "INFRA"); err != nil {
				return err
			}

			return logging.SetLevel(options.LogLevel)
		},
	}

	serverCmd, err := newServerCmd()
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newAccessCmd())
	rootCmd.AddCommand(newDestinationsCmd())
	rootCmd.AddCommand(newProvidersCmd())
	rootCmd.AddCommand(newMachinesCmd())
	rootCmd.AddCommand(newTokensCmd())
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(newEngineCmd())
	rootCmd.AddCommand(newVersionCmd())

	rootCmd.PersistentFlags().String("log-level", "info", "Set the log level. One of error, warn, info, or debug")

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
