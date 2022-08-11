package cmd

import (
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

	user, destinations, grants, err := getUserDestinationGrants(client)
	if err != nil {
		return err
	}

	gs := make(map[string]map[string]struct{})
	for _, g := range grants {
		// aggregate privileges
		if gs[g.Resource] == nil {
			gs[g.Resource] = make(map[string]struct{})
		}

		gs[g.Resource][g.Privilege] = struct{}{}
	}

	type row struct {
		Name   string `header:"NAME"`
		Access string `header:"ACCESS"`
	}

	var rows []row

	keys := make([]string, 0, len(gs))
	for k := range gs {
		if strings.HasPrefix(k, "infra") {
			continue
		}

		var exists bool
		for _, d := range destinations {
			if strings.HasPrefix(k, d.Name) {
				exists = true
				break
			}
		}

		if !exists {
			continue
		}

		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		v, ok := gs[k]
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

func getUserDestinationGrants(client *api.Client) (*api.User, []api.Destination, []api.Grant, error) {
	config, err := currentHostConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	if config.UserID == 0 {
		return nil, nil, nil, fmt.Errorf("no active identity")
	}

	user, err := client.GetUser(config.UserID)
	if err != nil {
		return nil, nil, nil, err
	}

	grants, err := listAll(client.ListGrants, api.ListGrantsRequest{User: config.UserID, ShowInherited: true})
	if err != nil {
		return nil, nil, nil, err
	}

	destinations, err := listAll(client.ListDestinations, api.ListDestinationsRequest{})
	if err != nil {
		return nil, nil, nil, err
	}

	return user, destinations, grants, nil
}
