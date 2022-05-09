package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newInfoCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Display the info about the current session",
		Args:  NoArgs,
		Group: "Other commands:",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := mustBeLoggedIn(); err != nil {
				return err
			}
			return info(cli)
		},
	}
}

func info(cli *CLI) error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cli.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Server:\t %s\n", config.Host)

	id := config.PolymorphicID
	if id == "" {
		return fmt.Errorf("no active identity")
	}

	identityID, err := id.ID()
	if err != nil {
		return err
	}

	identity, err := client.GetIdentity(identityID)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Identity:\t %s (%s)\n", identity.Name, identity.ID)

	if config.ProviderID != 0 {
		provider, err := client.GetProvider(config.ProviderID)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "Identity Provider:\t %s (%s)\n", provider.Name, provider.URL)
	}

	identityGroups, err := client.ListIdentityGroups(identityID)
	if err != nil {
		return err
	}

	groups := "(none)"

	if identityGroups.Count > 0 {
		g := make([]string, 0, identityGroups.Count)
		for _, identityGroup := range identityGroups.Items {
			g = append(g, identityGroup.Name)
		}

		groups = strings.Join(g, ", ")

	}

	fmt.Fprintf(w, "Groups:\t %s\n", groups)

	fmt.Fprintln(w)

	return nil
}
