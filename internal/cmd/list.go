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
		Group:   "Core commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cli)
		},
	}
}

func list(cli *CLI) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	user, destinations, grants, err := getUserDestinationGrants(client, "kubernetes")
	if err != nil {
		return err
	}

	grantsByResource := make(map[string]map[string]struct{})
	resources := []string{}
	for _, g := range grants {
		if isResourceForDestination(g.Resource, "infra") {
			continue
		}

		if !destinationForResourceExists(g.Resource, destinations) {
			continue
		}

		if grantsByResource[g.Resource] == nil {
			grantsByResource[g.Resource] = make(map[string]struct{})
			resources = append(resources, g.Resource)
		}

		grantsByResource[g.Resource][g.Privilege] = struct{}{}
	}
	sort.Strings(resources)

	type row struct {
		Name   string `header:"NAME"`
		Access string `header:"ACCESS"`
	}

	var rows []row
	for _, k := range resources {
		v, ok := grantsByResource[k]
		if !ok {
			// should not be possible
			return fmt.Errorf("unexpected value in grants: %s", k)
		}

		access := make([]string, 0, len(v))
		for vk := range v {
			access = append(access, vk)
		}

		rows = append(rows, row{
			Name:   k,
			Access: strings.Join(access, ", "),
		})
	}

	if len(rows) > 0 {
		printTable(rows, cli.Stdout)
	} else {
		cli.Output("You have not been granted access to any active destinations")
	}

	return writeKubeconfig(user, destinations, grants)
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
