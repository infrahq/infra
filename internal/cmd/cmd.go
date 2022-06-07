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
	"time"

	"github.com/goware/urlx"
	"github.com/lensesio/tableprinter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/connector"
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
			return Error{Message: "Not logged in; run 'infra login' before running this command"}
		}
		return fmt.Errorf("getting host config: %w", err)
	}

	if !config.isLoggedIn() {
		return Error{Message: "Not logged in; run 'infra login' before running this command"}
	}

	if config.isExpired() {
		return Error{Message: "Session expired; run 'infra login' to start a new session"}
	}

	return nil
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
	config, err := currentHostConfig()
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
			Timeout: 60 * time.Second,
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

func newUseCmd(_ *CLI) *cobra.Command {
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

			id, err := config.PolymorphicID.ID()
			if err != nil {
				return err
			}

			err = updateKubeconfig(client, id)
			if err != nil {
				return err
			}

			parts := strings.Split(destination, ".")

			if len(parts) == 1 {
				return kubernetesSetContext(destination, "")
			}

			return kubernetesSetContext(parts[0], parts[1])
		},
	}
}

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

func newConnectorCmd() *cobra.Command {
	var configFilename string

	cmd := &cobra.Command{
		Use:    "connector",
		Short:  "Start the Infra connector",
		Args:   NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logging.SetServerLogger()

			options := defaultConnectorOptions()
			err := cliopts.Load(&options, cliopts.Options{
				Filename:  configFilename,
				EnvPrefix: "INFRA_CONNECTOR",
				Flags:     cmd.Flags(),
			})
			if err != nil {
				return err
			}

			return runConnector(cmd.Context(), options)
		},
	}

	cmd.Flags().StringVarP(&configFilename, "config-file", "f", "", "Connector config file")
	cmd.Flags().StringP("server", "s", "", "Infra server hostname")
	cmd.Flags().StringP("access-key", "a", "", "Infra access key (use file:// to load from a file)")
	cmd.Flags().StringP("name", "n", "", "Destination name")
	cmd.Flags().String("ca-cert", "", "Path to CA certificate file")
	cmd.Flags().String("ca-key", "", "Path to CA key file")
	cmd.Flags().Bool("skip-tls-verify", false, "Skip verifying server TLS certificates")

	return cmd
}

// runConnector is a shim for testing
var runConnector = connector.Run

// defaultConnectorOptions is empty for now. It exists so that it can be
// referenced by a test.
func defaultConnectorOptions() connector.Options {
	return connector.Options{}
}

func NewRootCmd(cli *CLI) *cobra.Command {
	cobra.EnableCommandSorting = false

	rootCmd := &cobra.Command{
		Use:               "infra",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return rootPreRun(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Core commands:
	rootCmd.AddCommand(newLoginCmd(cli))
	rootCmd.AddCommand(newLogoutCmd(cli))
	rootCmd.AddCommand(newListCmd(cli))
	rootCmd.AddCommand(newUseCmd(cli))

	// Management commands:
	rootCmd.AddCommand(newDestinationsCmd(cli))
	rootCmd.AddCommand(newGrantsCmd(cli))
	rootCmd.AddCommand(newUsersCmd(cli))
	rootCmd.AddCommand(newGroupsCmd(cli))
	rootCmd.AddCommand(newKeysCmd(cli))
	rootCmd.AddCommand(newProvidersCmd(cli))

	// Other commands:
	rootCmd.AddCommand(newInfoCmd(cli))
	rootCmd.AddCommand(newVersionCmd(cli))

	// Hidden
	rootCmd.AddCommand(newTokensCmd(cli))
	rootCmd.AddCommand(newServerCmd())
	rootCmd.AddCommand(newConnectorCmd())
	rootCmd.AddCommand(newAgentCmd())

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
