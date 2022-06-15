package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

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

func newGroupsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Hidden:  true,
		Use:     "groups",
		Short:   "Manage groups of identities",
		Aliases: []string{"group"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newGroupsAddCmd(cli))
	cmd.AddCommand(newGroupsListCmd(cli))
	cmd.AddCommand(newGroupsRemoveCmd(cli))

	return cmd
}

func newGroupsListCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List groups",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			type row struct {
				Name  string `header:"Name"`
				Users string `header:"Users"`
			}

			var rows []row

			groups, err := client.ListGroups(api.ListGroupsRequest{})
			if err != nil {
				return err
			}

			for _, group := range groups.Items {
				users, err := client.ListUsers(api.ListUsersRequest{Group: group.ID})
				if err != nil {
					return err
				}

				var userNames []string
				for _, user := range users.Items {
					userNames = append(userNames, user.Name)
				}

				rows = append(rows, row{
					Name:  group.Name,
					Users: strings.Join(userNames, ", "),
				})
			}

			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No groups found")
			}

			return nil
		},
	}
}

func newGroupsAddCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "add GROUP",
		Short: "Create a group",
		Args:  ExactArgs(1),
		Example: `# Create a group
$ infra groups add Engineering`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			_, err = client.CreateGroup(&api.CreateGroupRequest{Name: args[0]})
			if err != nil {
				return err
			}
			cli.Output("Added group %q", args[0])

			return nil
		},
	}
}

func newGroupsRemoveCmd(cli *CLI) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "remove GROUP",
		Aliases: []string{"rm"},
		Short:   "Delete a group",
		Args:    ExactArgs(1),
		Example: `# Delete a group
$ infra groups remove Engineering`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			groups, err := client.ListGroups(api.ListGroupsRequest{Name: name})
			if err != nil {
				return err
			}

			if groups.Count == 0 && !force {
				return fmt.Errorf("unknown group %q", name)
			}

			for _, group := range groups.Items {
				if err := client.DeleteGroup(group.ID); err != nil {
					return err
				}

				cli.Output("Removed group %q", group.Name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Exit successfully even if the group does not exist")

	return cmd
}
