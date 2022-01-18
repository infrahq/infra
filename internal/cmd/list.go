package cmd

import (
	"fmt"
	"strings"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/uid"
)

type listRow struct {
	Name   string `header:"DESTINATION"`
	Access string `header:"ACCESS"`
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

	if config.ID == 0 {
		return fmt.Errorf("no active user")
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

	var rows []listRow

	for _, g := range grants {
		if g.Resource == "infra" {
			continue
		}

		rows = append(rows, listRow{
			Name:   g.Resource,
			Access: g.Privilege,
		})
	}

	printTable(rows)

	return updateKubeconfig()
}

func info(client *api.Client, g api.Grant) (provider string, name string, err error) {
	var id uid.ID

	switch {
	case strings.HasPrefix(g.Identity, "u:"):
		if err := id.UnmarshalText([]byte(strings.TrimPrefix(g.Identity, "u:"))); err != nil {
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
		if err := id.UnmarshalText([]byte(strings.TrimPrefix(g.Identity, "g:"))); err != nil {
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
