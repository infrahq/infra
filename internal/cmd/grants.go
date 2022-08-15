package cmd

import (
	"errors"
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
	UserName  string
	GroupName string
	Resource  string
	Role      string
	Force     bool
	Inherited bool
}

func newGrantsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "grants",
		Short:   "Manage access to resources",
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
			listReq := api.ListGrantsRequest{
				Privilege:     options.Role,
				Resource:      options.Resource,
				ShowInherited: options.Inherited,
			}

			if options.UserName != "" && options.GroupName != "" {
				return Error{Message: "You cannot use both a --user and a --group at the same time"}
			}

			if options.UserName != "" {
				user, err := getUserByNameOrID(client, options.UserName)
				if err != nil {
					return err
				}

				listReq.User = user.ID
			}

			if options.GroupName != "" {
				if options.Inherited {
					cli.Output("Warning: using --inherited with a group does nothing")
				}
				group, err := getGroupByNameOrID(client, options.GroupName)
				if err != nil {
					return err
				}

				listReq.Group = group.ID
			}

			grants, err := listAll(client.ListGrants, listReq)
			if err != nil {
				return err
			}

			numUserGrants, err := userGrants(cli, client, &grants)
			if err != nil {
				return err
			}

			if numUserGrants != 0 {
				cli.Output("")
			}

			numGroupGrants, err := groupGrants(cli, client, &grants)
			if err != nil {
				return err
			}

			if numUserGrants+numGroupGrants == 0 {
				cli.Output("No grants found")
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&options.Resource, "destination", "", "Filter by destination")
	cmd.Flags().StringVar(&options.GroupName, "group", "", "Filter by group name or id")
	cmd.Flags().StringVar(&options.UserName, "user", "", "Filter by user name or id")
	cmd.Flags().BoolVar(&options.Inherited, "inherited", false, "Include grants a user inherited through a group")
	cmd.Flags().StringVar(&options.Role, "role", "", "Filter by user role")
	return cmd
}

func userGrants(cli *CLI, client *api.Client, grants *[]api.Grant) (int, error) {
	users, err := listAll(client.ListUsers, api.ListUsersRequest{})
	if err != nil {
		return 0, err
	}

	mapUsers := make(map[uid.ID]api.User)
	for _, u := range users {
		mapUsers[u.ID] = u
	}

	items := slice.Select(*grants, func(g api.Grant) bool { return g.User != 0 })

	type row struct {
		User     string `header:"USER"`
		Role     string `header:"ROLE"`
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
			Role:     item.Privilege,
			Resource: item.Resource,
		})
	}

	if len(rows) > 0 {
		printTable(rows, cli.Stdout)
	}

	return len(rows), nil
}

func groupGrants(cli *CLI, client *api.Client, grants *[]api.Grant) (int, error) {
	groups, err := listAll(client.ListGroups, api.ListGroupsRequest{})
	if err != nil {
		return 0, err
	}

	mapGroups := make(map[uid.ID]api.Group)
	for _, u := range groups {
		mapGroups[u.ID] = u
	}

	items := slice.Select(*grants, func(g api.Grant) bool { return g.Group != 0 })

	type row struct {
		Group    string `header:"GROUP"`
		Role     string `header:"ROLE"`
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
			Role:     item.Privilege,
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
	var isGroup bool

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
			if isGroup {
				options.GroupName = args[0]
			} else {
				options.UserName = args[0]
			}
			options.Resource = args[1]
			return removeGrant(cli, options)
		},
	}

	cmd.Flags().BoolVarP(&isGroup, "group", "g", false, "Group to revoke access from")
	cmd.Flags().StringVar(&options.Role, "role", "", "Role to revoke")
	cmd.Flags().BoolVar(&options.Force, "force", false, "Exit successfully even if grant does not exist")

	return cmd
}

func removeGrant(cli *CLI, cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	user, group, err := checkUserGroup(client, cmdOptions.UserName, cmdOptions.GroupName)
	if err != nil {
		var cliError Error
		if errors.As(err, &cliError) {
			return Error{
				Message: fmt.Sprintf("Cannot revoke grants: %s", cliError.Message),
			}
		}
		return err
	}

	listGrantsReq := api.ListGrantsRequest{
		User:      user,
		Group:     group,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Resource,
	}

	logging.Debugf("call server: list grants %#v", listGrantsReq)
	grants, err := client.ListGrants(listGrantsReq)
	if err != nil {
		if api.ErrorStatusCode(err) == 403 {
			logging.Debugf("%s", err.Error())
			return Error{
				Message: "Cannot revoke grants: missing privileges for ListGrants",
			}
		}
		return err
	}

	if grants.Count == 0 && !cmdOptions.Force {
		return Error{Message: "Grant not found"}
	}

	for _, g := range grants.Items {
		logging.Debugf("call server: delete grant %s", g.ID)
		err := client.DeleteGrant(g.ID)
		if err != nil {
			if api.ErrorStatusCode(err) == 403 {
				logging.Debugf("%s", err.Error())
				return Error{
					Message: "Cannot revoke grants: missing privileges for DeleteGrant",
				}
			}
			if api.ErrorStatusCode(err) == 400 && strings.Contains(err.Error(), "cannot remove the last infra admin") {
				logging.Debugf("%s", err.Error())
				return Error{
					Message: "Cannot revoke grant: at least one Infra admin grant must exist",
				}
			}
			return err
		}

		if g.Group > 0 {
			cli.Output("Revoked %q access from group %q for resource %q", g.Privilege, cmdOptions.GroupName, g.Resource)
		} else if g.User > 0 {
			cli.Output("Revoked %q access from user %q for resource %q", g.Privilege, cmdOptions.UserName, g.Resource)
		}
	}

	return nil
}

func newGrantAddCmd(cli *CLI) *cobra.Command {
	var options grantsCmdOptions
	var isGroup bool

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
			if isGroup {
				options.GroupName = args[0]
			} else {
				options.UserName = args[0]
			}
			options.Resource = args[1]
			return addGrant(cli, options)
		},
	}

	cmd.Flags().BoolVarP(&isGroup, "group", "g", false, "When set, creates a grant for a group instead of a user")
	cmd.Flags().StringVar(&options.Role, "role", models.BasePermissionConnect, "Type of access that the user or group will be given")
	cmd.Flags().BoolVar(&options.Force, "force", false, "Create grant even if requested user, destination, or role are unknown")
	return cmd
}

func addGrant(cli *CLI, cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	userID, groupID, err := checkUserGroup(client, cmdOptions.UserName, cmdOptions.GroupName)
	if err != nil {
		var cliError Error
		if errors.As(err, &cliError) {
			return Error{
				Message: fmt.Sprintf("Cannot create grants: %s", cliError.Message),
			}
		}
		if !cmdOptions.Force {
			return err
		}
	}

	if userID == 0 && cmdOptions.UserName != "" {
		user, err := createUser(client, cmdOptions.UserName)
		if err != nil {
			if api.ErrorStatusCode(err) == 403 {
				logging.Debugf("%s", err.Error())
				return Error{
					Message: "Cannot create grants: missing privileges for ListGrants",
				}
			}
			return err
		}

		cli.Output("Created user %q", cmdOptions.UserName)
		userID = user.ID

	} else if groupID == 0 && cmdOptions.GroupName != "" {
		group, err := createGroup(client, cmdOptions.GroupName)
		if err != nil {

			if api.ErrorStatusCode(err) == 403 {
				logging.Debugf("%s", err.Error())
				return Error{
					Message: "Cannot create grants: missing privileges for CreateGroup",
				}
			}
			return err
		}

		cli.Output("Created group %q", cmdOptions.GroupName)
		groupID = group.ID
	}

	if err := checkResourcesPrivileges(client, cmdOptions.Resource, cmdOptions.Role); err != nil {
		if !cmdOptions.Force {
			return err
		}
	}

	createGrantReq := &api.CreateGrantRequest{
		User:      userID,
		Group:     groupID,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Resource,
	}
	logging.Debugf("call server: create grant %#v", createGrantReq)
	response, err := client.CreateGrant(createGrantReq)
	if err != nil {
		if api.ErrorStatusCode(err) == 403 {
			logging.Debugf("%s", err.Error())
			return Error{
				Message: "Cannot create grant: missing privileges for CreateGrant",
			}
		}
		return err
	}
	if response.WasCreated {
		cli.Output("Created grant to %q for %q", cmdOptions.Resource, cmdOptions.UserName+cmdOptions.GroupName)
	} else {
		cli.Output("%q grant to %q already exists for %q. Nothing changed", cmdOptions.Role, cmdOptions.Resource, cmdOptions.UserName+cmdOptions.GroupName)
	}

	return nil
}

// checkUserGroup returns the ID of the requested user or group if they exist. Otherwise it
// returns an error
func checkUserGroup(client *api.Client, user, group string) (userID uid.ID, groupID uid.ID, err error) {
	if group != "" {
		g, err := getGroupByNameOrID(client, group)
		if err != nil {
			if api.ErrorStatusCode(err) == 403 {
				logging.Debugf("%s", err.Error())
				return 0, 0, Error{
					Message: "missing privileges for GetGroup",
				}
			}
			return 0, 0, err
		}

		return 0, g.ID, nil
	}

	u, err := getUserByNameOrID(client, user)
	if err != nil {
		if api.ErrorStatusCode(err) == 403 {
			logging.Debugf("%s", err.Error())
			return 0, 0, Error{
				Message: "missing privileges for GetUser",
			}
		}
		return 0, 0, err
	}

	return u.ID, 0, nil
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
		logging.Debugf("call server: list destinations named %q", destination)
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
