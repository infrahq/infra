package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func getGroupByName(client *api.Client, name string) (*api.Group, error) {
	groups, err := client.ListGroups(api.ListGroupsRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if groups.Count == 0 {
		return nil, fmt.Errorf("%w: unknown group %q", ErrGroupNotFound, name)
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
	cmd.AddCommand(newGroupsAddUserCmd(cli))
	cmd.AddCommand(newGroupsListCmd(cli))
	cmd.AddCommand(newGroupsRemoveCmd(cli))
	cmd.AddCommand(newGroupsRemoveUserCmd(cli))

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

			groups, err := listAll(client, api.ListGroupsRequest{}, api.Client.ListGroups, nil)
			if err != nil {
				return err
			}

			for _, group := range groups {
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

func newGroupsAddUserCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "adduser USER GROUP",
		Short: "Add a user to a group",
		Args:  ExactArgs(2),
		Example: `# Add a user to a group
$ infra groups adduser johndoe@example.com Engineering
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userName := args[0]
			groupName := args[1]

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			user, err := getUserByName(client, userName)
			if err != nil {
				if errors.Is(err, ErrUserNotFound) {
					return Error{Message: fmt.Sprintf("unknown user %q", userName)}
				}
				return err
			}

			group, err := getGroupByName(client, groupName)
			if err != nil {
				if errors.Is(err, ErrGroupNotFound) {
					return Error{Message: fmt.Sprintf("unknown group %q", groupName)}
				}
				return err
			}

			req := &api.UpdateUsersInGroupRequest{
				GroupID:      group.ID,
				UserIDsToAdd: []uid.ID{user.ID},
			}
			err = client.UpdateUsersInGroup(req)
			if err != nil {
				return err
			}

			cli.Output("Added user %q to group %q", user.Name, group.Name)

			return nil
		},
	}
}

func newGroupsRemoveUserCmd(cli *CLI) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "removeuser USER GROUP",
		Short:   "Remove a user from a group",
		Aliases: []string{"rmuser"},
		Args:    ExactArgs(2),
		Example: `# Remove a user from a group
$ infra groups removeuser johndoe@example.com Engineering
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userName := args[0]
			groupName := args[1]

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			user, err := getUserByName(client, userName)
			if err != nil {
				if !force {
					if errors.Is(err, ErrUserNotFound) {
						return Error{Message: fmt.Sprintf("unknown user %q", userName)}
					}
					return err
				}
				return nil
			}

			group, err := getGroupByName(client, groupName)
			if err != nil {
				if !force {
					if errors.Is(err, ErrGroupNotFound) {
						return Error{Message: fmt.Sprintf("unknown group %q", groupName)}
					}
					return err
				}
				return nil
			}

			req := &api.UpdateUsersInGroupRequest{
				GroupID:         group.ID,
				UserIDsToRemove: []uid.ID{user.ID},
			}
			err = client.UpdateUsersInGroup(req)
			if err != nil {
				return err
			}

			cli.Output("Removed user %q from group %q", userName, groupName)

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Exit successfully even if the user or group does not exist")

	return cmd
}
