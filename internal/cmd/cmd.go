package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/lensesio/tableprinter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/logging"
)

// Run the main CLI command with the given args. The args should not contain
// the name of the binary (ex: os.Args[1:]).
func Run(ctx context.Context, args ...string) error {
	cli := newCLI(ctx)
	cmd := NewRootCmd(cli)
	cmd.SetArgs(args)
	cmd.SetErr(cli.Stderr)
	cmd.SetOut(cli.Stdout)
	return cmd.ExecuteContext(ctx)
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

type APIClientOpts struct {
	Host                     string
	AccessKey                string
	Transport                *http.Transport
	SkipLogoutOnUnauthorized bool
}

// Creates API Client options from the current config
func defaultClientOpts() (*APIClientOpts, error) {
	config, err := currentHostConfig()
	if err != nil {
		return nil, err
	}
	return apiClientFromHostConfig(config)
}

func apiClientFromHostConfig(config *ClientHostConfig) (*APIClientOpts, error) {
	server := config.Host
	var accessKey string
	if !config.isExpired() {
		accessKey = config.AccessKey
	}

	if envAccessKey, ok := os.LookupEnv("INFRA_ACCESS_KEY"); ok {
		accessKey = envAccessKey
	}

	if len(accessKey) == 0 {
		if config.isExpired() {
			return nil, Error{Message: "Access key is expired, please `infra login` again", OriginalError: ErrAccessKeyExpired}
		}
		return nil, Error{Message: "Missing access key, must `infra login` or set INFRA_ACCESS_KEY in your environment", OriginalError: ErrAccessKeyMissing}
	}

	if envServer, ok := os.LookupEnv("INFRA_SERVER"); ok {
		server = envServer
	}

	return &APIClientOpts{
		Host:      server,
		AccessKey: accessKey,
		Transport: httpTransportForHostConfig(config),
	}, nil
}

func NewAPIClient(opts *APIClientOpts) (*api.Client, error) {
	if opts.Host == "" || opts.Transport == nil {
		return nil, fmt.Errorf("api client access key, host, and transport are required")
	}
	client := &api.Client{
		Name:      "cli",
		Version:   internal.Version,
		URL:       "https://" + opts.Host,
		AccessKey: opts.AccessKey,
		HTTP: http.Client{
			Timeout:   60 * time.Second,
			Transport: opts.Transport,
		},
	}
	if !opts.SkipLogoutOnUnauthorized {
		client.OnUnauthorized = logoutCurrent
	}
	return client, nil
}

func logoutCurrent() {
	config, err := readConfig()
	if err != nil {
		logging.Debugf("logging out: read config: %s", err)
		return
	}

	var host *ClientHostConfig
	for i := range config.Hosts {
		if config.Hosts[i].Current {
			host = &config.Hosts[i]
			break
		}
	}

	if host == nil {
		return
	}

	host.AccessKey = ""
	host.Expires = api.Time{}
	host.UserID = 0
	host.Name = ""

	if err := writeConfig(config); err != nil {
		logging.Debugf("logging out: write config: %s", err)
		return
	}
}

func httpTransportForHostConfig(config *ClientHostConfig) *http.Transport {
	pool, err := x509.SystemCertPool()
	if err != nil {
		logging.Warnf("Failed to load trusted certificates from system: %v", err)
		pool = x509.NewCertPool()
	}

	if config.TrustedCertificate != "" {
		ok := pool.AppendCertsFromPEM([]byte(config.TrustedCertificate))
		if !ok {
			logging.Warnf("Failed to read trusted certificates for server")
		}
	}

	return &http.Transport{
		TLSClientConfig: &tls.Config{
			//nolint:gosec // We may purposely set insecureskipverify via a flag
			InsecureSkipVerify: config.SkipTLSVerify,
			RootCAs:            pool,
		},
	}
}

const (
	groupCore       = "group-core"
	groupManagement = "group-management"
	groupServices   = "group-services"
)

func NewRootCmd(cli *CLI) *cobra.Command {
	cobra.EnableCommandSorting = false

	var showAdminHelp bool

	rootCmd := &cobra.Command{
		Use:               "infra",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := cliopts.DefaultsFromEnv("INFRA", cmd.Flags()); err != nil {
				return err
			}
			if err := logging.SetLevel(cli.RootOptions.LogLevel); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showAdminHelp {
				cmd.SetUsageFunc(nil) // hide our custom template
				fmt.Fprintln(cli.Stdout, cmd.UsageString())
				return nil
			}
			return nil
		},
	}

	rootCmd.AddGroup(
		&cobra.Group{
			ID:    groupCore,
			Title: "Core commands:",
		},
		&cobra.Group{
			ID:    groupManagement,
			Title: "Admin commands:",
		},
		&cobra.Group{
			ID:    groupServices,
			Title: "Service commands:",
		})

	rootCmd.AddCommand(
		// Core commands
		newLoginCmd(cli),
		newLogoutCmd(cli),
		newListCmd(cli),
		newUseCmd(cli),

		// Management commands
		newDestinationsCmd(cli),
		newGrantsCmd(cli),
		newUsersCmd(cli),
		newGroupsCmd(cli),
		newKeysCmd(cli),
		newProvidersCmd(cli),

		// Other commands
		newInfoCmd(cli),
		newVersionCmd(cli),

		// Hidden commands
		newTokensCmd(cli),
		newServerCmd(),
		newConnectorCmd(),
		newAgentCmd(),
		newSSHCmd(cli),
		newSSHDCmd(cli))

	rootCmd.PersistentFlags().Bool("help", false, "Display help")
	rootCmd.PersistentFlags().StringVar(&cli.RootOptions.LogLevel, "log-level", "info", "Show logs when running the command [error, warn, info, debug]")
	rootCmd.PersistentFlags().BoolVar(&cli.RootOptions.SkipAPIVersionCheck, "skip-version-check", false, "Skip checking if the CLI is ahead of the server version")
	rootCmd.Flags().BoolVar(&showAdminHelp, "help-admin", false, "Show help for admin commands")

	rootCmd.AddCommand(newAboutCmd())
	rootCmd.AddCommand(newCompletionsCmd())
	rootCmd.SetUsageFunc(usageFunc(rootCmd))
	return rootCmd
}

func addNonInteractiveFlag(flags *pflag.FlagSet, bind *bool) {
	isNonInteractiveMode := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))
	flags.BoolVar(bind, "non-interactive", isNonInteractiveMode, "Disable all prompts for input")
}

func addFormatFlag(flags *pflag.FlagSet, bind *string) {
	flags.StringVar(bind, "format", "", "Output format [json|yaml]")
}

func usageFunc(rootCmd *cobra.Command) func(cmd *cobra.Command) error {
	orig := rootCmd.UsageFunc()

	return func(cmd *cobra.Command) error {
		// Use the default text for non-root commands
		if cmd != rootCmd {
			return orig(cmd)
		}

		cmd.SetUsageTemplate(rootUsage)
		return orig(cmd)
	}
}

// Modified from spf13/cobra.Command.UsageTemplate
var rootUsage = `Usage:
  infra [command]

{{- $cmds := .Commands }}

Core commands:{{range $cmds}}
  {{- if (and (eq .GroupID "group-core") .IsAvailableCommand )}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

{{- if not .AllChildCommandsHaveGroup}}

Other commands:{{range $cmds}}{{if (and (eq .GroupID "") .IsAvailableCommand )}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`
