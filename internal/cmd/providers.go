package cmd

import (
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

type providerOptions struct {
	URL          string
	ClientID     string `mapstructure:"client-id"`
	ClientSecret string `mapstructure:"client-secret"`
}

func newProvidersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List connected identity providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			providers, err := client.ListProviders("")
			if err != nil {
				return err
			}

			type row struct {
				Name string `header:"NAME"`
				URL  string `header:"URL"`
			}

			var rows []row
			for _, p := range providers {
				rows = append(rows, row{Name: p.Name, URL: p.URL})
			}

			printTable(rows)

			return nil
		},
	}
}

func newProvidersAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add NAME URL CLIENT_ID CLIENT_SECRET",
		Short: "Connect an identity provider",
		Long: `
Add an identity provider for users to authenticate.

NAME: The name of the identity provider (e.g. okta)
URL: The base URL of the domain used to login with the identity provider (e.g. acme.okta.com)
CLIENT_ID: The Infra application OpenID Connect client ID
CLIENT_SECRET: The Infra application OpenID Connect client secret
		`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options providerOptions
			if err := parseOptions(cmd, &options, "INFRA_PROVIDER"); err != nil {
				return err
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			_, err = client.CreateProvider(&api.CreateProviderRequest{
				Name:         args[0],
				URL:          args[1],
				ClientID:     args[2],
				ClientSecret: args[3],
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func newProvidersRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove PROVIDER",
		Aliases: []string{"rm"},
		Short:   "Disconnect an identity provider",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			providers, err := client.ListProviders(args[0])
			if err != nil {
				return err
			}

			for _, p := range providers {
				err := client.DeleteProvider(p.ID)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
