package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goware/urlx"
	"github.com/iancoleman/strcase"
	"github.com/lensesio/tableprinter"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/connector"
	"github.com/infrahq/infra/internal/decode"
	"github.com/infrahq/infra/internal/logging"
)

// Run the main CLI command with the given args. The args should not contain
// the name of the binary (ex: os.Args[1:]).
func Run(ctx context.Context, args ...string) error {
	cli := newCLI(ctx)
	cmd := NewRootCmd(cli)
	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}

func mustBeLoggedIn() error {
	config, err := currentHostConfig()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return fmt.Errorf("Not logged in. Run 'infra login' before running this command.")
		}
		return fmt.Errorf("getting host config: %w", err)
	}

	if !config.isLoggedIn() {
		return fmt.Errorf("Not logged in. Run 'infra login' before running this command.")
	}

	if config.isExpired() {
		return fmt.Errorf("Session expired. Run 'infra login' to start a new session.")
	}

	return nil
}

func parseOptions(cmd *cobra.Command, options interface{}, envPrefix string) error {
	v := viper.New()

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

	if err := v.ReadInConfig(); err != nil {
		var errNotFound *viper.ConfigFileNotFoundError
		if errors.As(err, &errNotFound) {
			return err
		}
	}

	hooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		decode.HookPrepareForDecode,
		decode.HookSetFromString,
	)
	return v.Unmarshal(options, viper.DecodeHook(hooks))
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

func printTable(data interface{}, out io.Writer) {
	table := tableprinter.New(out)

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

// Creates a new API Client from the current config
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

	headers := http.Header{}
	ua := fmt.Sprintf("Infra CLI/%v (%v/%v)", internal.Version, runtime.GOOS, runtime.GOARCH)
	headers.Add("User-Agent", ua)

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
		Headers: headers,
	}, nil
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use DESTINATION",
		Short: "Access a destination",
		Example: `
# Use a Kubernetes context
$ infra use development

# Use a Kubernetes namespace context
$ infra use development.kube-system`,
		Args:  ExactArgs(1),
		Group: "Core commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			destination := args[0]

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

			parts := strings.Split(destination, ".")

			if parts[0] == "kubernetes" {
				if len(parts) > 2 {
					return kubernetesSetContext(parts[1], parts[2])
				}

				return kubernetesSetContext(parts[1], "")
			}

			// no type specifier, guess at user intent
			if len(parts) == 1 {
				return kubernetesSetContext(destination, "")
			}

			return kubernetesSetContext(parts[0], parts[1])
		},
	}
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

func newConnectorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "connector",
		Short:  "Start the Infra connector",
		Args:   NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logging.SetServerLogger()

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

			return connector.Run(cmd.Context(), options)
		},
	}

	cmd.Flags().StringP("config-file", "f", "", "Connector config file")
	cmd.Flags().StringP("server", "s", "", "Infra server hostname")
	cmd.Flags().StringP("access-key", "a", "", "Infra access key (use file:// to load from a file)")
	cmd.Flags().StringP("name", "n", "", "Destination name")
	cmd.Flags().String("tls-cert", "$HOME/.infra/cache/tls.crt", "Path to TLS certificate file")
	cmd.Flags().String("tls-key", "$HOME/.infra/cache/tls.key", "Path to TLS key file")
	cmd.Flags().String("tls-cache", "$HOME/.infra/cache", "Directory to cache TLS certificates")
	cmd.Flags().Bool("skip-tls-verify", false, "Skip verifying server TLS certificates")

	return cmd
}

// rootOptions are options specified by users on the command line that are
// used by the root command.
type rootOptions struct {
	Info    bool
	Version bool
}

func NewRootCmd(cli *CLI) *cobra.Command {
	cobra.EnableCommandSorting = false
	var rootOpts rootOptions

	rootCmd := &cobra.Command{
		Use:               "infra",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return rootPreRun(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if rootOpts.Version {
				return version()
			}
			if rootOpts.Info {
				if err := mustBeLoggedIn(); err != nil {
					return fmt.Errorf("login check: %w", err)
				}
				if err := info(); err != nil {
					return fmt.Errorf("info: %w", err)
				}
				return nil
			}
			return cmd.Help()
		},
	}

	// Core commands:
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newUseCmd())

	// Management commands:
	rootCmd.AddCommand(newDestinationsCmd())
	rootCmd.AddCommand(newGrantsCmd())
	rootCmd.AddCommand(newIdentitiesCmd())
	rootCmd.AddCommand(newKeysCmd(cli))
	rootCmd.AddCommand(newProvidersCmd())

	// Hidden
	rootCmd.AddCommand(newTokensCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newServerCmd())
	rootCmd.AddCommand(newConnectorCmd())
	rootCmd.AddCommand(newVersionCmd())

	rootCmd.Flags().BoolVar(&rootOpts.Version, "version", false, "Display Infra version")
	rootCmd.Flags().BoolVar(&rootOpts.Info, "info", false, "Display info about the current logged in session")

	rootCmd.PersistentFlags().String("log-level", "info", "Show logs when running the command [error, warn, info, debug]")
	rootCmd.PersistentFlags().Bool("help", false, "Display help")

	rootCmd.SetHelpCommandGroup("Other commands:")
	rootCmd.AddCommand(newAboutCmd())
	rootCmd.SetUsageTemplate(usageTemplate())
	return rootCmd
}

func rootPreRun(flags *pflag.FlagSet) error {
	if err := cliopts.DefaultsFromEnv("INFRA", flags); err != nil {
		return err
	}
	logLevel, err := flags.GetString("log-level")
	if err != nil {
		return err
	}
	if err := logging.SetLevel(logLevel); err != nil {
		return err
	}
	return nil
}

func addNonInteractiveFlag(flags *pflag.FlagSet, bind *bool) {
	isNonInteractiveMode := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))
	flags.BoolVar(bind, "non-interactive", isNonInteractiveMode, "Disable all prompts for input")
}

func usageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}
  
Available Commands:{{end}}{{range $cmds}}{{if (and (eq .Group "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .Group $group.Group) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}
