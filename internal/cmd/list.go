package cmd

import (
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/infrahq/infra/internal/api"
)

func list() error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	if config.ID == 0 {
		return fmt.Errorf("no active identity")
	}

	destinations, err := client.ListDestinations(api.ListDestinationsRequest{})
	if err != nil {
		return err
	}

	var grants []api.Grant
	if config.ProviderID != 0 {

		grants, err = client.ListUserGrants(config.ID)
		if err != nil {
			return err
		}

		groups, err := client.ListUserGroups(config.ID)
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
	} else {
		grants, err = client.ListMachineGrants(config.ID)
		if err != nil {
			return err
		}
	}

	gs := make(map[string]mapset.Set)
	for _, g := range grants {
		// aggregate privileges
		if gs[g.Resource] == nil {
			gs[g.Resource] = mapset.NewSet()
		}

		gs[g.Resource].Add(g.Privilege)
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

		rows = append(rows, row{
			Name:   k,
			Access: v.String()[4 : len(v.String())-1],
		})
	}

	printTable(rows)

	return writeKubeconfig(destinations, grants)
}

func info(client *api.Client, g api.Grant) (provider string, name string, err error) {
	id, err := g.Identity.ID()
	if err != nil {
		return "", "", err
	}

	switch {
	case g.Identity.IsUser():
		user, err := client.GetUser(id)
		if err != nil {
			return "", "", err
		}

		provider, err := client.GetProvider(user.ProviderID)
		if err != nil {
			return "", "", err
		}

		return provider.Name, user.Email, nil
	case g.Identity.IsMachine():
		machine, err := client.GetMachine(id)
		if err != nil {
			return "", "", err
		}

		return "", machine.Name, nil
	default:
		group, err := client.GetGroup(id)
		if err != nil {
			return "", "", err
		}

		provider, err := client.GetProvider(group.ProviderID)
		if err != nil {
			return "", "", err
		}

		return provider.Name, group.Name, nil
	}
}
