package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func getGroupByNameOrID(client *api.Client, name string) (*api.Group, error) {
	ctx := context.TODO()

	req := api.ListGroupsRequest{Name: name}
	groups, err := client.ListGroups(ctx, req)
	if err != nil {
		return nil, err
	}

	if groups.Count == 0 {
		if id, err := uid.Parse([]byte(name)); err == nil {
			g, err := client.GetGroup(ctx, id)
			if err == nil {
				return g, nil
			}
		}

		return nil, fmt.Errorf("%w: unknown group %q", ErrGroupNotFound, name)
	}

	if groups.Count > 1 {
		return nil, fmt.Errorf("multiple results found for %q. check your server configurations", name)
	}

	return &groups.Items[0], nil
}

func newGroupsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "groups",
		Short:   "Manage groups of identities",
		Aliases: []string{"group"},
		GroupID: groupManagement,
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
	var noTruncate bool
	var numUsers int
	cmd := &cobra.Command{
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
				Name      string `header:"Name"`
				Users     string `header:"Users"`
				UserCount int    `header:"Count"`
			}

			ctx := context.Background()

			groups, err := listAll(ctx, client.ListGroups, api.ListGroupsRequest{})
			if err != nil {
				return err
			}

			var rows []row
			for _, group := range groups {
				var users []api.User
				if noTruncate {
					users, err = listAll(ctx, client.ListUsers, api.ListUsersRequest{Group: group.ID})
					if err != nil {
						return err
					}
				} else if numUsers != 0 {
					userRes, err := client.ListUsers(ctx, api.ListUsersRequest{
						PaginationRequest: api.PaginationRequest{Limit: numUsers},
						Group:             group.ID,
					})
					if err != nil {
						return err
					}
					users = userRes.Items
				}

				var userNames []string
				for _, user := range users {
					userNames = append(userNames, user.Name)
				}

				if !noTruncate && group.TotalUsers > numUsers {
					userNames = append(userNames, "...")
				}

				rows = append(rows, row{
					Name:      group.Name,
					Users:     strings.Join(userNames, ", "),
					UserCount: group.TotalUsers,
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
	cmd.Flags().BoolVar(&noTruncate, "no-truncate", false, "Do not truncate the list of users for each group")
	cmd.Flags().IntVar(&numUsers, "num-users", 8, "The number of users to display in each group")
	return cmd
}

func newGroupsAddCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "add GROUP",
		Short: "Create a group",
		Args:  ExactArgs(1),
		Example: `# Create a group
$ infra groups add Engineering`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			_, err = client.CreateGroup(ctx, &api.CreateGroupRequest{Name: args[0]})
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

			ctx := context.Background()

			groups, err := client.ListGroups(ctx, api.ListGroupsRequest{Name: name})
			if err != nil {
				return err
			}

			if groups.Count == 0 && !force {
				return fmt.Errorf("unknown group %q", name)
			}

			for _, group := range groups.Items {
				if err := client.DeleteGroup(ctx, group.ID); err != nil {
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

			ctx := context.Background()

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			user, err := getUserByNameOrID(client, userName)
			if err != nil {
				if errors.Is(err, ErrUserNotFound) {
					return Error{Message: fmt.Sprintf("unknown user %q", userName)}
				}
				return err
			}

			group, err := getGroupByNameOrID(client, groupName)
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
			err = client.UpdateUsersInGroup(ctx, req)
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

			ctx := context.Background()

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			user, err := getUserByNameOrID(client, userName)
			if err != nil {
				if !force {
					if errors.Is(err, ErrUserNotFound) {
						return Error{Message: fmt.Sprintf("unknown user %q", userName)}
					}
					return err
				}
				return nil
			}

			group, err := getGroupByNameOrID(client, groupName)
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
			err = client.UpdateUsersInGroup(ctx, req)
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
