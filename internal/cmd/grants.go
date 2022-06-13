package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ssoroka/slice"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type grantsCmdOptions struct {
	Name        string
	Destination string
	IsGroup     bool
	Role        string
	Force       bool
}

func newGrantsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "grants",
		Short:   "Manage access to destinations",
		Aliases: []string{"grant"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newGrantsListCmd(cli))
	cmd.AddCommand(newGrantAddCmd(cli))
	cmd.AddCommand(newGrantRemoveCmd(cli))

	return cmd
}

func newGrantsListCmd(cli *CLI) *cobra.Command {
	var options grantsCmdOptions

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List grants",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			grants, err := client.ListGrants(api.ListGrantsRequest{Resource: options.Destination})
			if err != nil {
				return err
			}

			numUserGrants, err := userGrants(cli, client, grants)
			if err != nil {
				return err
			}

			numGroupGrants, err := groupGrants(cli, client, grants)
			if err != nil {
				return err
			}

			if numUserGrants+numGroupGrants == 0 {
				cli.Output("No grants found")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&options.Destination, "destination", "", "Filter by destination")
	return cmd
}

func userGrants(cli *CLI, client *api.Client, grants *api.ListResponse[api.Grant]) (int, error) {
	users, err := client.ListUsers(api.ListUsersRequest{})
	if err != nil {
		return 0, err
	}

	mapUsers := make(map[uid.ID]api.User)
	for _, u := range users.Items {
		mapUsers[u.ID] = u
	}

	items := slice.Select(grants.Items, func(g api.Grant) bool { return g.User != 0 })

	type row struct {
		User     string `header:"USER"`
		Access   string `header:"ACCESS"`
		Resource string `header:"DESTINATION"`
	}

	rows := make([]row, 0, len(items))
	for _, item := range items {
		user, ok := mapUsers[item.User]
		if !ok {
			return 0, fmt.Errorf("unknown user for ID %v", item.ID)
		}

		rows = append(rows, row{
			User:     user.Name,
			Access:   item.Privilege,
			Resource: item.Resource,
		})
	}

	if len(rows) > 0 {
		printTable(rows, cli.Stdout)
	}

	return len(rows), nil
}

func groupGrants(cli *CLI, client *api.Client, grants *api.ListResponse[api.Grant]) (int, error) {
	groups, err := client.ListGroups(api.ListGroupsRequest{})
	if err != nil {
		return 0, err
	}

	mapGroups := make(map[uid.ID]api.Group)
	for _, u := range groups.Items {
		mapGroups[u.ID] = u
	}

	items := slice.Select(grants.Items, func(g api.Grant) bool { return g.Group != 0 })

	type row struct {
		Group    string `header:"GROUP"`
		Access   string `header:"ACCESS"`
		Resource string `header:"DESTINATION"`
	}

	rows := make([]row, 0, len(items))
	for _, item := range items {
		group, ok := mapGroups[item.Group]
		if !ok {
			return 0, fmt.Errorf("unknown group for ID %v", item.ID)
		}

		rows = append(rows, row{
			Group:    group.Name,
			Access:   item.Privilege,
			Resource: item.Resource,
		})
	}

	if len(rows) > 0 {
		printTable(rows, cli.Stdout)
	}

	return len(rows), nil
}

func newGrantRemoveCmd(cli *CLI) *cobra.Command {
	var options grantsCmdOptions

	cmd := &cobra.Command{
		Use:     "remove USER|GROUP DESTINATION",
		Aliases: []string{"rm"},
		Short:   "Revoke a user or group's access to a destination",
		Example: `# Remove all grants of a user in a destination
$ infra grants remove janedoe@example.com docker-desktop

# Remove all grants of a group in a destination
$ infra grants remove group-a staging --group

# Remove a specific grant
$ infra grants remove janedoe@example.com staging --role viewer

# Remove adminaccess to infra
$ infra grants remove janedoe@example.com infra --role admin
`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Name = args[0]
			options.Destination = args[1]
			return removeGrant(cli, options)
		},
	}

	cmd.Flags().BoolVarP(&options.IsGroup, "group", "g", false, "Group to revoke access from")
	cmd.Flags().StringVar(&options.Role, "role", "", "Role to revoke")
	cmd.Flags().BoolVar(&options.Force, "force", false, "Exit successfully even if grant does not exist")

	return cmd
}

func removeGrant(cli *CLI, cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	user, group, err := checkUserGroup(client, cmdOptions.Name, cmdOptions.IsGroup)
	if err != nil {
		return err
	}

	listGrantsReq := api.ListGrantsRequest{
		User:      user,
		Group:     group,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Destination,
	}

	logging.S.Debugf("call server: list grants %#v", listGrantsReq)
	grants, err := client.ListGrants(listGrantsReq)
	if err != nil {
		return err
	}

	if grants.Count == 0 && !cmdOptions.Force {
		return Error{Message: "Grant not found"}
	}

	for _, g := range grants.Items {
		logging.S.Debugf("call server: delete grant %s", g.ID)
		err := client.DeleteGrant(g.ID)
		if err != nil {
			return err
		}

		cli.Output("Revoked access from user %q for destination %q", cmdOptions.Name, cmdOptions.Destination)
	}

	return nil
}

func newGrantAddCmd(cli *CLI) *cobra.Command {
	var options grantsCmdOptions

	cmd := &cobra.Command{
		Use:   "add USER|GROUP DESTINATION",
		Short: "Grant a user or group access to a destination",
		Example: `# Grant a user access to a destination
$ infra grants add johndoe@example.com docker-desktop

# Grant a group access to a destination
$ infra grants add group-a staging --group

# Grant access with fine-grained permissions
$ infra grants add johndoe@example.com staging --role viewer

# Assign a user a role within Infra
$ infra grants add johndoe@example.com infra --role admin
`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Name = args[0]
			options.Destination = args[1]
			return addGrant(cli, options)
		},
	}

	cmd.Flags().BoolVarP(&options.IsGroup, "group", "g", false, "When set, creates a grant for a group instead of a user")
	cmd.Flags().StringVar(&options.Role, "role", models.BasePermissionConnect, "Type of access that the user or group will be given")
	cmd.Flags().BoolVar(&options.Force, "force", false, "Create grant even if requested user, destination, or role are unknown")
	return cmd
}

func addGrant(cli *CLI, cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	userID, groupID, err := checkUserGroup(client, cmdOptions.Name, cmdOptions.IsGroup)
	if err != nil {
		if !cmdOptions.Force {
			return err
		}
	}

	if userID == 0 && !cmdOptions.IsGroup {
		user, err := createUser(client, cmdOptions.Name, false)
		if err != nil {
			return err
		}

		cli.Output("Created user %q", cmdOptions.Name)
		userID = user.ID

	} else if groupID == 0 && cmdOptions.IsGroup {
		group, err := createGroup(client, cmdOptions.Name)
		if err != nil {
			return err
		}

		cli.Output("Created group %q", cmdOptions.Name)
		groupID = group.ID
	}

	if err := checkResourcesPrivileges(client, cmdOptions.Destination, cmdOptions.Role); err != nil {
		if !cmdOptions.Force {
			return err
		}
	}

	createGrantReq := &api.CreateGrantRequest{
		User:      userID,
		Group:     groupID,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Destination,
	}
	logging.S.Debugf("call server: create grant %#v", createGrantReq)
	_, err = client.CreateGrant(createGrantReq)
	if err != nil {
		return err
	}

	cli.Output("Created grant to %q for %q", cmdOptions.Destination, cmdOptions.Name)

	return nil
}

// checkUserGroup returns the ID of the requested user or group if they exist. Otherwise it
// returns an error
func checkUserGroup(client *api.Client, subject string, isGroup bool) (userID uid.ID, groupID uid.ID, err error) {
	if isGroup {
		group, err := getGroupByName(client, subject)
		if err != nil {
			return 0, 0, err
		}

		return 0, group.ID, nil
	}

	user, err := getUserByName(client, subject)
	if err != nil {
		return 0, 0, err
	}

	return user.ID, 0, nil
}

// checkResourcesPrivileges checks if the requested destination (e.g. cluster), optional
// resource (e.g. namespace), and role exist. destination "infra" and role "connect" are
// reserved values and will always pass checks
func checkResourcesPrivileges(client *api.Client, resource, privilege string) error {
	parts := strings.SplitN(resource, ".", 2)
	destination := parts[0]
	subresource := ""

	if len(parts) > 1 {
		subresource = parts[1]
	}

	supportedResources := make(map[string]struct{})
	supportedRoles := make(map[string]struct{})

	if destination != "infra" {
		logging.S.Debugf("call server: list destinations named %q", destination)
		destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: destination})
		if err != nil {
			return err
		}

		if destinations.Count == 0 {
			return Error{Message: fmt.Sprintf("Destination %q not connected; to ignore, run with '--force'", destination)}
		}

		for _, d := range destinations.Items {
			for _, r := range d.Resources {
				supportedResources[r] = struct{}{}
			}

			for _, r := range d.Roles {
				supportedRoles[r] = struct{}{}
			}
		}

		if subresource != "" {
			if _, ok := supportedResources[subresource]; !ok {
				return Error{Message: fmt.Sprintf("Namespace %q not detected in destination %q; to ignore, run with '--force'", subresource, destination)}
			}
		}

		if privilege != "connect" {
			if _, ok := supportedRoles[privilege]; !ok {
				return Error{Message: fmt.Sprintf("Role %q is not a known role for destination %q; to ignore, run with '--force'", privilege, destination)}
			}
		}
	}

	return nil
}
