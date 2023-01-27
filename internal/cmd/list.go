package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

func newListCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List accessible destinations",
		Args:    NoArgs,
		GroupID: groupCore,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cli)
		},
	}
}

func list(cli *CLI) error {
	client, err := cli.apiClient()
	if err != nil {
		return err
	}

	_, destinations, grants, err := getUserDestinationGrants(client, "")
	if err != nil {
		return err
	}

	type row struct {
		Name     string `header:"NAME"`
		Resource string `header:"RESOURCE"`
		Access   string `header:"ROLE"`
	}

	rows := make([]row, 0, len(grants))
	for _, g := range grants {
		if g.DestinationName == "infra" {
			continue
		}

		if !destinationForResourceExists(g.DestinationName, destinations) {
			continue
		}

		rows = append(rows, row{
			Name:     g.DestinationName,
			Resource: g.DestinationResource,
			Access:   g.Privilege,
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}

		if rows[i].Resource != rows[j].Resource {
			return rows[i].Resource < rows[j].Resource
		}

		return rows[i].Access < rows[j].Access
	})

	if len(rows) > 0 {
		printTable(rows, cli.Stdout)
	} else {
		cli.Output("You have not been granted access to any active destinations")
	}

	return updateKubeconfig(client)
}

func destinationForResourceExists(resource string, destinations []api.Destination) bool {
	for _, d := range destinations {
		if !isResourceForDestination(resource, d.Name) {
			continue
		}

		return isDestinationAvailable(d)
	}

	return false
}

func isDestinationAvailable(destination api.Destination) bool {
	return destination.Connected && destination.Connection.URL != ""
}

func isResourceForDestination(resource string, destination string) bool {
	return resource == destination || strings.HasPrefix(resource, destination+".")
}

func getUserDestinationGrants(client *api.Client, kind string) (*api.User, []api.Destination, []api.Grant, error) {
	ctx := context.TODO()

	config, err := currentHostConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	if config.UserID == 0 {
		return nil, nil, nil, fmt.Errorf("no active identity")
	}

	user, err := client.GetUser(ctx, config.UserID)
	if err != nil {
		return nil, nil, nil, err
	}

	grants, err := listAll(ctx, client.ListGrants, api.ListGrantsRequest{User: config.UserID, ShowInherited: true})
	if err != nil {
		return nil, nil, nil, err
	}

	destinations, err := listAll(ctx, client.ListDestinations, api.ListDestinationsRequest{Kind: kind})
	if err != nil {
		return nil, nil, nil, err
	}

	return user, destinations, grants, nil
}
