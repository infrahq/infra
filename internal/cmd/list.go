package cmd

import (
	"fmt"
	"strings"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/uid"
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
		return fmt.Errorf("no active user")
	}

	destinations, err := client.ListDestinations(api.ListDestinationsRequest{})
	if err != nil {
		return err
	}

	grants, err := client.ListUserGrants(config.ID)
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

	gs := make(map[string]string)
	for _, g := range grants {
		// aggregate privileges
		gs[g.Resource] = gs[g.Resource] + g.Privilege + " "
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
			Access: v,
		})
	}

	printTable(rows)

	return writeKubeconfig(destinations, grants)
}

func info(client *api.Client, g api.Grant) (provider string, name string, err error) {
	switch {
	case strings.HasPrefix(g.Identity, "u:"):
		id, err := uid.ParseString(strings.TrimPrefix(g.Identity, "u:"))
		if err != nil {
			return "", "", err
		}

		user, err := client.GetUser(id)
		if err != nil {
			return "", "", err
		}

		provider, err := client.GetProvider(user.ProviderID)
		if err != nil {
			return "", "", err
		}

		return provider.Name, user.Email, nil
	default:
		id, err := uid.ParseString(strings.TrimPrefix(g.Identity, "g:"))
		if err != nil {
			return "", "", err
		}

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
