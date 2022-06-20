package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/logging"
)

func newProvidersCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "providers",
		Short:   "Manage identity providers",
		Aliases: []string{"provider"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newProvidersListCmd(cli))
	cmd.AddCommand(newProvidersAddCmd(cli))
	cmd.AddCommand(newProvidersRemoveCmd(cli))

	return cmd
}

func newProvidersListCmd(cli *CLI) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List connected identity providers",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			logging.S.Debug("call server: list providers")
			providers, err := client.ListProviders("")
			if err != nil {
				return err
			}

			switch format {
			case "json":
				jsonOutput, err := json.Marshal(providers)
				if err != nil {
					return err
				}
				cli.Output(string(jsonOutput))
			default:
				type row struct {
					Name string `header:"NAME"`
					Kind string `header:"KIND"`
					URL  string `header:"URL"`
				}

				var rows []row
				for _, p := range providers.Items {
					rows = append(rows, row{Name: p.Name, URL: p.URL, Kind: p.Kind})
				}

				if len(rows) > 0 {
					printTable(rows, cli.Stdout)
				} else {
					cli.Output("No providers found")
				}
			}
			return nil
		},
	}

	addFormatFlag(cmd.Flags(), &format)
	return cmd
}

type providerAddOptions struct {
	URL          string
	ClientID     string
	ClientSecret string
	Kind         string
}

func (o providerAddOptions) Validate() error {
	var missing []string
	if o.URL == "" {
		missing = append(missing, "url")
	}
	if o.ClientID == "" {
		missing = append(missing, "client-id")
	}
	if o.ClientSecret == "" {
		missing = append(missing, "client-secret")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing value for required flags: %v", strings.Join(missing, ", "))
	}
	return nil
}

func newProvidersAddCmd(cli *CLI) *cobra.Command {
	var opts providerAddOptions

	cmd := &cobra.Command{
		Use:   "add PROVIDER",
		Short: "Connect an identity provider",
		Long: `Add an identity provider for users to authenticate.
PROVIDER is a short unique name of the identity provider being added (eg. okta)`,
		Example: `# Connect okta to infra
$ infra providers add okta --url example.okta.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cliopts.DefaultsFromEnv("INFRA_PROVIDER", cmd.Flags()); err != nil {
				return err
			}

			if err := opts.Validate(); err != nil {
				return err
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			logging.S.Debugf("call server: create provider named %q", args[0])
			_, err = client.CreateProvider(&api.CreateProviderRequest{
				Name:         args[0],
				URL:          opts.URL,
				ClientID:     opts.ClientID,
				ClientSecret: opts.ClientSecret,
				Kind:         opts.Kind,
			})
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.S.Debug(err)
					return Error{
						Message: "Cannot connect provider: missing privileges for CreateProvider",
					}
				}
				return err
			}

			cli.Output("Connected provider %q (%s) to infra", args[0], opts.URL)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.URL, "url", "", "Base URL of the domain of the OIDC identity provider (eg. acme.okta.com)")
	cmd.Flags().StringVar(&opts.ClientID, "client-id", "", "OIDC client ID")
	cmd.Flags().StringVar(&opts.ClientSecret, "client-secret", "", "OIDC client secret")
	cmd.Flags().StringVar(&opts.Kind, "kind", "oidc", "The identity provider kind. One of 'oidc, okta, azure, or google'")
	return cmd
}

func newProvidersRemoveCmd(cli *CLI) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "remove PROVIDER",
		Aliases: []string{"rm"},
		Short:   "Disconnect an identity provider",
		Example: "$ infra providers remove okta",
		Args:    ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			logging.S.Debugf("call server: list providers named %q", args[0])
			providers, err := client.ListProviders(args[0])
			if err != nil {
				return err
			}

			if providers.Count == 0 && !force {
				return Error{Message: fmt.Sprintf("No identity providers connected with the name %q", args[0])}
			}

			logging.S.Debugf("deleting %s providers named %q...", providers.Count, args[0])
			for _, provider := range providers.Items {
				logging.S.Debugf("...call server: delete provider %s", provider.ID)
				if err := client.DeleteProvider(provider.ID); err != nil {
					if api.ErrorStatusCode(err) == 403 {
						logging.S.Debug(err)
						return Error{
							Message: "Cannot disconnect provider: missing privileges for DeleteProvider",
						}
					}
					return err
				}

				cli.Output("Disconnected provider %q (%s) from infra", provider.Name, provider.URL)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Exit successfully even if provider does not exist")

	return cmd
}

func GetProviderByName(client *api.Client, name string) (*api.Provider, error) {
	logging.S.Debugf("call server: list providers named %q", name)
	providers, err := client.ListProviders(name)
	if err != nil {
		return nil, err
	}

	if providers.Count == 0 {
		return nil, Error{Message: fmt.Sprintf("No identity providers connected with the name %q", name)}
	}

	return &providers.Items[0], nil
}
