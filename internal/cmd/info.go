package cmd

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

func newInfoCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "info",
		Short:   "Display the info about the current session",
		Args:    NoArgs,
		GroupID: groupOther,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return info(cli)
		},
	}
}

func info(cli *CLI) error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := cli.apiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	user, err := client.GetUser(ctx, config.UserID)
	if err != nil {
		if api.ErrorStatusCode(err) == 401 {
			return Error{Message: "Session is not valid for this server; run 'infra login' to start a new session"}
		}
		return err
	}

	w := tabwriter.NewWriter(cli.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Server:\t %s\n", config.Host)

	if config.UserID == 0 {
		return fmt.Errorf("no active user")
	}

	fmt.Fprintf(w, "User:\t %s (%s)\n", user.Name, user.ID)

	if config.ProviderID != 0 {
		provider, err := client.GetProvider(ctx, config.ProviderID)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "Identity Provider:\t %s (%s)\n", provider.Name, provider.URL)
	}

	userGroups, err := listAll(ctx, client.ListGroups, api.ListGroupsRequest{UserID: config.UserID})
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

	fmt.Fprintln(w)

	return nil
}
