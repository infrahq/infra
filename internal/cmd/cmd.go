package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/goware/urlx"
	"github.com/iancoleman/strcase"
	"github.com/lensesio/tableprinter"
	"github.com/mcuadros/go-defaults"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/connector"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server"
)

func mustBeLoggedIn() error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	if !config.isLoggedIn() {
		return fmt.Errorf("Not logged in. Run 'infra login' before running this command.")
	}

	return nil
}

func parseOptions(cmd *cobra.Command, options interface{}, envPrefix string) error {
	v := viper.New()

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

	if envPrefix != "" {
		v.SetEnvPrefix(envPrefix)
		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		v.AutomaticEnv()

		// workaround for viper not correctly binding env vars
		// https://github.com/spf13/viper/issues/761
		envKeys := make(map[string]interface{})
		if err := mapstructure.Decode(options, &envKeys); err != nil {
			return err
		}

		// bind file options (lower camel case) to environment options (envPrefix + upper snake case)
		// e.g. accessKey -> INFRA_CONNECTOR_ACCESS_KEY
		for envKey := range envKeys {
			fullEnvKey := fmt.Sprintf("%s_%s", envPrefix, envKey)
			if err := v.BindEnv(envKey, strcase.ToScreamingSnake(fullEnvKey)); err != nil {
				return err
			}
		}
	}

	errs := make([]error, 0)
	// bind command line options (lower snake case) to file options (lower camel case)
	// e.g. access-key -> accessKey
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err := v.BindPFlag(strcase.ToLowerCamel(f.Name), f); err != nil {
			errs = append(errs, err)
		}
	})

	if len(errs) > 0 {
		var sb strings.Builder

		sb.WriteString("multiple errors seen while binding flags:\n\n")

		for _, err := range errs {
			fmt.Fprintf(&sb, "* %s\n", err)
		}

		return errors.New(sb.String())
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

	return apiClient(config.Host, config.AccessKey, config.SkipTLSVerify)
}

func apiClient(host string, accessKey string, skipTLSVerify bool) (*api.Client, error) {
	u, err := urlx.Parse(host)
	if err != nil {
		return nil, err
	}

	u.Scheme = "https"

	return &api.Client{
		URL:       fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		AccessKey: accessKey,
		HTTP: http.Client{
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
		Short:   "Login to Infra",
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
	var force bool

	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "Logout of Infra",
		Example: "$ infra logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logout(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "logout and remove context")

	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List accessible destinations",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return list()
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use DESTINATION",
		Short: "Access a destination",
		Example: `
# Connect to a Kubernetes cluster
$ infra use kubernetes.development

# Connect to a Kubernetes namespace
$ infra use kubernetes.development.kube-system
		`,
		Args: cobra.ExactArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
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

			err = updateKubeconfig(client, config.PolymorphicID)
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

func newGrantsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "grants",
		Short:   "Manage access to destinations",
		Aliases: []string{"grant"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newGrantsListCmd())
	cmd.AddCommand(newGrantAddCmd())
	cmd.AddCommand(newGrantRemoveCmd())

	return cmd
}

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Short:   "Manage access keys for machine identities to authenticate with Infra and call the API",
		Aliases: []string{"key"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newKeysListCmd())
	cmd.AddCommand(newKeysAddCmd())
	cmd.AddCommand(newKeysRemoveCmd())

	return cmd
}

func newDestinationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destinations",
		Aliases: []string{"dst", "dest", "destination"},
		Short:   "Manage destinations",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newDestinationsListCmd())
	cmd.AddCommand(newDestinationsAddCmd())
	cmd.AddCommand(newDestinationsRemoveCmd())

	return cmd
}

func canonicalPath(in string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	out := in
	if strings.HasPrefix(in, "$HOME") {
		out = strings.Replace(in, "$HOME", homeDir, 1)
	} else if strings.HasPrefix(in, "~") {
		out = strings.Replace(in, "~", homeDir, 1)
	}

	abs, err := filepath.Abs(out)
	if err != nil {
		return "", err
	}

	return abs, nil
}

func newOpenAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "openapi",
		Short:  "generate the openapi spec",
		Hidden: true,

		RunE: func(cmd *cobra.Command, args []string) error {
			s := &server.Server{}
			s.GenerateRoutes()
			return nil
		},
	}

	return cmd
}

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the Infra server",

		RunE: func(cmd *cobra.Command, args []string) error {
			// override default strcase.ToLowerCamel behaviour
			strcase.ConfigureAcronym("enable-ui", "enableUI")
			strcase.ConfigureAcronym("ui-proxy-url", "uiProxyURL")

			options := server.Options{}
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

			return server.Run(options)
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
	cmd.Flags().DurationP("session-duration", "d", time.Hour*12, "User session duration")
	cmd.Flags().Bool("enable-setup", true, "Enable one-time setup")

	return cmd
}

func newConnectorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connector",
		Short: "Start the Infra connector",
		RunE: func(cmd *cobra.Command, args []string) error {
			// override default strcase.ToLowerCamel behaviour
			strcase.ConfigureAcronym("skip-tls-verify", "skipTLSVerify")

			var options connector.Options
			if err := parseOptions(cmd, &options, "INFRA_CONNECTOR"); err != nil {
				return err
			}

			tlsCache, err := canonicalPath(options.TLSCache)
			if err != nil {
				return err
			}

			options.TLSCache = tlsCache

			return connector.Run(options)
		},
	}

	cmd.Flags().StringP("config-file", "f", "", "Connector config file")
	cmd.Flags().StringP("server", "s", "", "Infra server hostname")
	cmd.Flags().StringP("access-key", "a", "", "Infra access key (use file:// to load from a file)")
	cmd.Flags().StringP("name", "n", "", "Destination name")
	cmd.Flags().String("tls-cache", "$HOME/.infra/cache", "Directory to cache TLS certificates")
	cmd.Flags().Bool("skip-tls-verify", false, "Skip verifying server TLS certificates")

	return cmd
}

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tokens",
		Short:  "Create & manage tokens",
		Hidden: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newTokensAddCmd())

	return cmd
}

func newProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "providers",
		Short:   "Manage identity providers",
		Aliases: []string{"provider"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newProvidersListCmd())
	cmd.AddCommand(newProvidersAddCmd())
	cmd.AddCommand(newProvidersRemoveCmd())

	return cmd
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "info",
		Short:  "Display the info about the current session",
		Hidden: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return info()
		},
	}
}

func newIdentitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identities",
		Aliases: []string{"id", "identity"},
		Short:   "Manage identities (users & machines)",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newIdentitiesAddCmd())
	cmd.AddCommand(newIdentitiesEditCmd())
	cmd.AddCommand(newIdentitiesListCmd())
	cmd.AddCommand(newIdentitiesRemoveCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "version",
		Short:  "Display the Infra version",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return version()
		},
	}
}

var nonInteractiveMode bool

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	type rootOptions struct {
		LogLevel       string `mapstructure:"logLevel"`
		NonInteractive bool   `mapstructure:"nonInteractive"`
	}

	var (
		versionFlag bool
		infoFlag    bool
	)

	rootCmd := &cobra.Command{
		Use:               "infra",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var options rootOptions
			if err := parseOptions(cmd, &options, "INFRA"); err != nil {
				return err
			}

			nonInteractiveMode = options.NonInteractive

			return logging.SetLevel(options.LogLevel)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if versionFlag {
				return version()
			}
			if infoFlag {
				if err := mustBeLoggedIn(); err != nil {
					return err
				}
				return info()
			}
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newGrantsCmd())
	rootCmd.AddCommand(newKeysCmd())
	rootCmd.AddCommand(newDestinationsCmd())
	rootCmd.AddCommand(newProvidersCmd())
	rootCmd.AddCommand(newIdentitiesCmd())
	rootCmd.AddCommand(newTokensCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newServerCmd())
	rootCmd.AddCommand(newOpenAPICmd())
	rootCmd.AddCommand(newConnectorCmd())
	rootCmd.AddCommand(newVersionCmd())

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Display Infra version")
	rootCmd.Flags().BoolVarP(&infoFlag, "info", "i", false, "Display info about the current session")

	rootCmd.PersistentFlags().String("log-level", "info", "Set the log level. One of error, warn, info, or debug")
	rootCmd.PersistentFlags().Bool("non-interactive", false, "don't assume an interactive terminal, even if there is one")

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	err = cmd.Execute()
	printError(err)

	return err
}

func printError(err error) {
	if err != nil {
		if !errors.Is(err, terminal.InterruptErr) {
			fmt.Fprintln(os.Stderr, "error: "+err.Error())
		}
	}
}
