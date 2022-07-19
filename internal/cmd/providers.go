package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/cmd/types"
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
	cmd.AddCommand(newProvidersEditCmd(cli))
	cmd.AddCommand(newProvidersRemoveCmd(cli))

	return cmd
}

type providerAPIOptions struct {
	PrivateKey  string
	ClientEmail string
	DomainAdmin string
}

func (o providerAPIOptions) Validate(providerKind string) error {
	if providerKind != "google" {
		// Parameters to configure API calls are currently only applicable to Google
		inapplicableFields := []string{}
		if o.ClientEmail != "" {
			inapplicableFields = append(inapplicableFields, "clientEmail")
		}
		if o.DomainAdmin != "" {
			inapplicableFields = append(inapplicableFields, "domainAdmin")
		}
		if o.PrivateKey != "" {
			inapplicableFields = append(inapplicableFields, "privateKey")
		}

		if len(inapplicableFields) > 0 {
			return fmt.Errorf("field(s) %q are only applicable to Google identity providers", inapplicableFields)
		}
	}
	return nil
}

type providerEditOptions struct {
	ClientSecret       string
	ProviderAPIOptions providerAPIOptions
}

func (o providerEditOptions) Validate(providerKind string) error {
	if o.ClientSecret == "" && o.ProviderAPIOptions.PrivateKey == "" && o.ProviderAPIOptions.ClientEmail == "" && o.ProviderAPIOptions.DomainAdmin == "" {
		return fmt.Errorf("Please specify a field to update.'\n\n%s", newProvidersEditCmd(nil).UsageString())
	}

	if providerKind != "google" && o.ClientSecret == "" {
		return fmt.Errorf("Client secret flag must be specified when updating an identity provider that isn't of kind Google")
	}

	return o.ProviderAPIOptions.Validate(providerKind)
}

func newProvidersEditCmd(cli *CLI) *cobra.Command {
	var opts providerEditOptions

	cmd := &cobra.Command{
		Use:   "edit PROVIDER",
		Short: "Update a provider",
		Example: `# Set a new client secret for a connected provider
$ infra providers edit okta --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN

# Connect Google to Infra with group sync
$ infra providers edit google --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --service-account--key ~/client-123.json --service-account--email hello@example.com --domain-admin admin@example.com
`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateProvider(cli, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.ClientSecret, "client-secret", "", "Set a new client secret")
	cmd.Flags().Var((*types.StringOrFile)(&opts.ProviderAPIOptions.PrivateKey), "service-account--key", "The private key used to make authenticated requests to Google's API")
	cmd.Flags().StringVar(&opts.ProviderAPIOptions.ClientEmail, "service-account--email", "", "The email assigned to the Infra service client in Google")
	cmd.Flags().StringVar(&opts.ProviderAPIOptions.DomainAdmin, "domain-admin", "", "The email of your Google workspace domain admin")
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

			logging.Debugf("call server: list providers")
			providers, err := listAll(client.ListProviders, api.ListProvidersRequest{})
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
			case "yaml":
				yamlOutput, err := yaml.Marshal(providers)
				if err != nil {
					return err
				}
				cli.Output(string(yamlOutput))
			default:
				type row struct {
					Name string `header:"NAME"`
					Kind string `header:"KIND"`
					URL  string `header:"URL"`
				}

				var rows []row
				for _, p := range providers {
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
	URL                string
	ClientID           string
	ClientSecret       string
	Kind               string
	ProviderAPIOptions providerAPIOptions
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
	return o.ProviderAPIOptions.Validate(o.Kind)
}

func newProvidersAddCmd(cli *CLI) *cobra.Command {
	var opts providerAddOptions

	cmd := &cobra.Command{
		Use:   "add PROVIDER",
		Short: "Connect an identity provider",
		Long: `Add an identity provider for users to authenticate.
PROVIDER is a short unique name of the identity provider being added (eg. okta)`,
		Example: `# Connect Okta to Infra
$ infra providers add okta --url example.okta.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --kind okta

# Connect Google to Infra with group sync
$ infra providers add google --url accounts.google.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --service-account--key ~/client-123.json --service-account--email hello@example.com --domain-admin admin@example.com --kind google`,
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

			privateKey, err := parsePrivateKey(opts.ProviderAPIOptions.PrivateKey)
			if err != nil {
				return err
			}

			logging.Debugf("call server: create provider named %q", args[0])
			_, err = client.CreateProvider(&api.CreateProviderRequest{
				Name:         args[0],
				URL:          opts.URL,
				ClientID:     opts.ClientID,
				ClientSecret: opts.ClientSecret,
				Kind:         opts.Kind,
				API: &api.ProviderAPICredentials{
					PrivateKey:  api.PEM(privateKey),
					ClientEmail: opts.ProviderAPIOptions.ClientEmail,
					DomainAdmin: opts.ProviderAPIOptions.DomainAdmin,
				},
			})
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.Debugf("%s", err.Error())
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
	cmd.Flags().Var((*types.StringOrFile)(&opts.ProviderAPIOptions.PrivateKey), "service-account--key", "The private key used to make authenticated requests to Google's API")
	cmd.Flags().StringVar(&opts.ProviderAPIOptions.ClientEmail, "service-account--email", "", "The email assigned to the Infra service client in Google")
	cmd.Flags().StringVar(&opts.ProviderAPIOptions.DomainAdmin, "domain-admin", "", "The email of your Google workspace domain admin")
	return cmd
}

func updateProvider(cli *CLI, name string, opts providerEditOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	res, err := client.ListProviders(api.ListProvidersRequest{Name: name})
	if err != nil {
		return err
	}

	if res.Count == 0 {
		return Error{
			Message: fmt.Sprintf("Provider %s does not exist", name),
		}
	}
	provider := res.Items[0]

	if err := opts.ProviderAPIOptions.Validate(provider.Kind); err != nil {
		return err
	}

	privateKey, err := parsePrivateKey(opts.ProviderAPIOptions.PrivateKey)
	if err != nil {
		return err
	}

	logging.Debugf("call server: update provider named %q", name)
	_, err = client.UpdateProvider(api.UpdateProviderRequest{
		ID:           provider.ID,
		Name:         name,
		URL:          provider.URL,
		ClientID:     provider.ClientID,
		ClientSecret: opts.ClientSecret,
		Kind:         provider.Kind,
		API: &api.ProviderAPICredentials{
			PrivateKey:  api.PEM(privateKey),
			ClientEmail: opts.ProviderAPIOptions.ClientEmail,
			DomainAdmin: opts.ProviderAPIOptions.DomainAdmin,
		},
	})

	if err != nil {
		if api.ErrorStatusCode(err) == 403 {
			logging.Debugf("%v", err)
			return Error{
				Message: "Cannot update provider: missing privileges for UpdateProvider",
			}
		}
		return err
	}

	return nil
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

			logging.Debugf("call server: list providers named %q", args[0])
			providers, err := client.ListProviders(api.ListProvidersRequest{Name: args[0]})
			if err != nil {
				return err
			}

			if providers.Count == 0 && !force {
				return Error{Message: fmt.Sprintf("No identity providers connected with the name %q", args[0])}
			}

			logging.Debugf("deleting %d providers named %q...", providers.Count, args[0])
			for _, provider := range providers.Items {
				logging.Debugf("...call server: delete provider %s", provider.ID)
				if err := client.DeleteProvider(provider.ID); err != nil {
					if api.ErrorStatusCode(err) == 403 {
						logging.Debugf("%s", err.Error())
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
	logging.Debugf("call server: list providers named %q", name)
	providers, err := client.ListProviders(api.ListProvidersRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if providers.Count == 0 {
		return nil, Error{Message: fmt.Sprintf("No identity providers connected with the name %q", name)}
	}

	return &providers.Items[0], nil
}

func parsePrivateKey(key string) (string, error) {
	if json.Valid([]byte(key)) {
		// this is the google service account key file
		jsonContents := map[string]string{}
		json.Unmarshal([]byte(key), &jsonContents)

		if jsonContents["private_key"] == "" {
			return "", fmt.Errorf("invalid service account json file provided")
		}

		// overwrite the full private key file with just the key
		return jsonContents["private_key"], nil
	}
	return key, nil
}
