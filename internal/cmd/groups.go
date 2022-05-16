package cmd

import (
	"fmt"

	"github.com/infrahq/infra/api"
)

func getGroupByName(client *api.Client, name string) (*api.Group, error) {
	groups, err := client.ListGroups(api.ListGroupsRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if groups.Count == 0 {
		return nil, fmt.Errorf("unknown group %q", name)
	}

	if groups.Count > 1 {
		return nil, fmt.Errorf("multiple results found for %q. check your server configurations", name)
	}

	return &groups.Items[0], nil
}

// createGroup creates a group with the requested name
func createGroup(client *api.Client, name string) (*api.Group, error) {
	group, err := client.CreateGroup(&api.CreateGroupRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return group, nil
}
