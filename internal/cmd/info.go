package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/server/models"
)

func newInfoCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:    "info",
		Short:  "Display the info about the current session",
		Args:   NoArgs,
		Hidden: true,
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

	if identity.Kind == models.UserKind.String() {
		userGroups, err := client.ListIdentityGroups(identityID)
		if err != nil {
			return err
		}

		groups := "(none)"

		if len(userGroups) > 0 {
			g := make([]string, 0, len(userGroups))
			for _, userGroup := range userGroups {
				g = append(g, userGroup.Name)
			}

			groups = strings.Join(g, ", ")

		}

		fmt.Fprintf(w, "Groups:\t %s\n", groups)
	}

	fmt.Fprintln(w)

	return nil
}
