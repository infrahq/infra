package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List accessible destinations",
		Group:   "Core commands:",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return list()
		},
	}
}

func list() error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	id := config.PolymorphicID
	if id == "" {
		return fmt.Errorf("no active identity")
	}

	identityID, err := id.ID()
	if err != nil {
		return err
	}

	grants, err := client.ListIdentityGrants(identityID)
	if err != nil {
		return err
	}

	groups, err := client.ListIdentityGroups(identityID)
	if err != nil {
		return err
	}

	for _, g := range groups {
		groupGrants, err := client.ListGroupGrants(g.ID)
		if err != nil {
			return err
		}

		grants = append(grants, groupGrants...)
	}

	gs := make(map[string]map[string]struct{})
	for _, g := range grants {
		// aggregate privileges
		if gs[g.Resource] == nil {
			gs[g.Resource] = make(map[string]struct{})
		}

		gs[g.Resource][g.Privilege] = struct{}{}
	}

	destinations, err := client.ListDestinations(api.ListDestinationsRequest{})
	if err != nil {
		return err
	}

	type row struct {
		Name   string `header:"RESOURCE"`
		Access string `header:"ACCESS"`
	}

	var rows []row

	for k, v := range gs {
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
		printTable(rows)
	} else {
		fmt.Printf("No grants found for user %q\n", config.Name)
	}

	return writeKubeconfig(destinations, grants)
}

func subjectNameFromGrant(client *api.Client, g api.Grant) (name string, err error) {
	id, err := g.Subject.ID()
	if err != nil {
		return "", err
	}

	if g.Subject.IsIdentity() {
		identity, err := client.GetIdentity(id)
		if err != nil {
			return "", err
		}

		return identity.Name, nil
	}

	if g.Subject.IsGroup() {
		group, err := client.GetGroup(id)
		if err != nil {
			return "", err
		}

		return group.Name, nil
	}

	return "", fmt.Errorf("unrecognized grant subject")
}
