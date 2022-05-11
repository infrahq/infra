package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type grantsCmdOptions struct {
	Identity    string
	Destination string
	IsGroup     bool
	Role        string
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

			type userRow struct {
				User     string `header:"USER"`
				Access   string `header:"ACCESS"`
				Resource string `header:"DESTINATION"`
			}
			type groupRow struct {
				Group    string `header:"GROUP"`
				Access   string `header:"ACCESS"`
				Resource string `header:"DESTINATION"`
			}

			var userRows []userRow
			var groupRows []groupRow
			for _, g := range grants.Items {

				switch {
				case g.User != 0:
					identity, err := client.GetUser(g.User)
					if err != nil {
						return err
					}
					userRows = append(userRows, userRow{
						User:     identity.Name,
						Access:   g.Privilege,
						Resource: g.Resource,
					})
				case g.Group != 0:
					group, err := client.GetGroup(g.Group)
					if err != nil {
						return err
					}

					groupRows = append(groupRows, groupRow{
						Group:    group.Name,
						Access:   g.Privilege,
						Resource: g.Resource,
					})
				default:
					// unknown grant subject
					continue
				}
			}

			if len(userRows)+len(groupRows) == 0 {
				cli.Output("No grants found")
				return nil
			}

			if len(userRows) > 0 {
				printTable(userRows, cli.Stdout)
			}
			if len(groupRows) > 0 {
				if len(userRows) > 0 {
					cli.Output("")
				}
				printTable(groupRows, cli.Stdout)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&options.Destination, "destination", "", "Filter by destination")
	return cmd
}

func newGrantRemoveCmd(cli *CLI) *cobra.Command {
	var options grantsCmdOptions

	cmd := &cobra.Command{
		Use:     "remove IDENTITY DESTINATION",
		Aliases: []string{"rm"},
		Short:   "Revoke an identity's access from a destination",
		Example: `# Remove all grants of an identity in a destination
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
			options.Identity = args[0]
			options.Destination = args[1]
			return removeGrant(cli, options)
		},
	}

	cmd.Flags().BoolVarP(&options.IsGroup, "group", "g", false, "Group to revoke access from")
	cmd.Flags().StringVar(&options.Role, "role", "", "Role to revoke")
	return cmd
}

func removeGrant(cli *CLI, cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	user, group, err := userOrGroupByName(client, cmdOptions.Identity, cmdOptions.IsGroup)
	if err != nil {
		return err
	}

	grants, err := client.ListGrants(api.ListGrantsRequest{
		User:      user,
		Group:     group,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Destination,
	})
	if err != nil {
		return err
	}

	for _, g := range grants.Items {
		err := client.DeleteGrant(g.ID)
		if err != nil {
			return err
		}
	}

	cli.Output("Access revoked!")

	return nil
}

func newGrantAddCmd(cli *CLI) *cobra.Command {
	var options grantsCmdOptions

	cmd := &cobra.Command{
		Use:   "add IDENTITY DESTINATION",
		Short: "Grant an identity access to a destination",
		Example: `# Grant an identity access to a destination
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
			options.Identity = args[0]
			options.Destination = args[1]
			return addGrant(cli, options)
		},
	}

	cmd.Flags().BoolVarP(&options.IsGroup, "group", "g", false, "Required if identity is of type 'group'")
	cmd.Flags().StringVar(&options.Role, "role", models.BasePermissionConnect, "Type of access that identity will be given")
	return cmd
}

func addGrant(cli *CLI, cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	user, group, err := userOrGroupByName(client, cmdOptions.Identity, cmdOptions.IsGroup)
	if err != nil {
		if !errors.Is(err, ErrUserNotFound) {
			return err
		}

		if cmdOptions.IsGroup {
			group, err = addGrantGroup(client, cmdOptions.Identity)
			if err != nil {
				return err
			}

		} else {
			user, err = addGrantUser(client, cmdOptions.Identity)
			if err != nil {
				return err
			}
		}
	}

	if err := checkResourcesPrivileges(client, cmdOptions.Destination, cmdOptions.Role); err != nil {
		return err
	}

	_, err = client.CreateGrant(&api.CreateGrantRequest{
		User:      user,
		Group:     group,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Destination,
	})
	if err != nil {
		return err
	}

	cli.Output("Access granted!")

	return nil
}

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
		destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: destination})
		if err != nil {
			return err
		}

		if destinations.Count == 0 {
			return fmt.Errorf("unknown destination %q", destination)
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
				return fmt.Errorf("unknown resource %q for %q", subresource, destination)
			}
		}

		if privilege != "connect" {
			if _, ok := supportedRoles[privilege]; !ok {
				return fmt.Errorf("unknown role %q for %q", privilege, destination)
			}
		}
	}

	return nil
}

// identityOrGroupByName gets the ID of the identity or group to be associated with the grant
func userOrGroupByName(client *api.Client, subject string, isGroup bool) (uid.ID, uid.ID, error) {
	if isGroup {
		group, err := GetGroupByName(client, subject)
		if err != nil {
			return 0, 0, err
		}

		return 0, group.ID, nil
	}

	user, err := GetUserByName(client, subject)
	if err != nil {
		return 0, 0, err
	}

	return user.ID, 0, nil
}

func GetGroupByName(client *api.Client, name string) (*api.Group, error) {
	groups, err := client.ListGroups(api.ListGroupsRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if groups.Count == 0 {
		return nil, ErrUserNotFound
	}

	if groups.Count != 1 {
		return nil, fmt.Errorf("invalid groups response, there should only be one group that matches a name, but multiple were found")
	}

	return &groups.Items[0], nil
}

func addGrantUser(client *api.Client, name string) (uid.ID, error) {
	identity, err := CreateUser(&api.CreateUserRequest{Name: name})
	if err != nil {
		return 0, err
	}

	fmt.Printf("New unlinked user %q added to Infra\n", name)

	return identity.ID, nil
}

func addGrantGroup(client *api.Client, name string) (uid.ID, error) {
	group, err := client.CreateGroup(&api.CreateGroupRequest{Name: name})
	if err != nil {
		return 0, err
	}

	fmt.Printf("New group %q added to Infra\n", name)

	return group.ID, nil
}
