package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/server/models"
)

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

func info() error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	id := config.PolymorphicID
	if id == "" {
		return fmt.Errorf("no active identity")
	}

	identityID, err := id.ID()
	if err != nil {
		return err
	}

	provider, err := client.GetProvider(config.ProviderID)
	if err != nil {
		return err
	}

	identity, err := client.GetIdentity(identityID)
	if err != nil {
		return err
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Server:\t", config.Host)
	fmt.Fprintf(w, "Identity Provider:\t %s (%s)\n", provider.Name, provider.URL)
	fmt.Fprintln(w, "Identity:\t", identity.Name)

	if identity.Kind == models.UserKind.String() {
		userGroups, err := client.ListIdentityGroups(identityID)
		if err != nil {
			return err
		}

		groups := make([]string, 0)
		for _, g := range userGroups {
			groups = append(groups, g.Name)
		}

		fmt.Fprintln(w, "Groups:\t", strings.Join(groups, ", "))
	}

	fmt.Fprintln(w)

	return nil
}
