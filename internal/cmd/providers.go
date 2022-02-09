package cmd

import (
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/api"
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
		Short:   "List identity providers",
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
		Use:   "add NAME",
		Short: "Connect an identity provider",
		Args:  cobra.ExactArgs(1),
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
				URL:          options.URL,
				ClientID:     options.ClientID,
				ClientSecret: options.ClientSecret,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("url", "", "url or domain (e.g. acme.okta.com)")
	cmd.Flags().String("client-id", "", "OpenID Client ID")
	cmd.Flags().String("client-secret", "", "OpenID Client Secret")

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
