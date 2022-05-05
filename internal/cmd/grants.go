package cmd

import (
	"errors"
	"fmt"

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

			type row struct {
				Identity string `header:"IDENTITY"`
				Access   string `header:"ACCESS"`
				Resource string `header:"DESTINATION"`
			}

			var rows []row
			for _, g := range grants.Items {
				identity, err := subjectNameFromGrant(client, g)
				if err != nil {
					return err
				}

				rows = append(rows, row{
					Identity: identity,
					Access:   g.Privilege,
					Resource: g.Resource,
				})
			}

			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No grants found")
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
$ infra grants remove janedoe@example.com kubernetes.docker-desktop 

# Remove all grants of a group in a destination
$ infra grants remove group-a kubernetes.staging --group

# Remove a specific grant 
$ infra grants remove janedoe@example.com kubernetes.staging --role viewer

# Remove access to infra 
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

	pid, err := getSubjectPolymorphicID(client, cmdOptions.Identity, cmdOptions.IsGroup)
	if err != nil {
		return err
	}

	grants, err := client.ListGrants(api.ListGrantsRequest{
		Subject:   pid,
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
$ infra grants add johndoe@example.com kubernetes.docker-desktop 

# Grant a group access to a destination 
$ infra grants add group-a kubernetes.staging --group

# Grant access with fine-grained permissions
$ infra grants add johndoe@example.com kubernetes.staging --role viewer

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

	pid, err := getSubjectPolymorphicID(client, cmdOptions.Identity, cmdOptions.IsGroup)
	if err != nil {
		if !errors.Is(err, ErrIdentityNotFound) {
			return err
		}
		if cmdOptions.IsGroup {
			pid, err = addGrantGroup(client, cmdOptions.Identity)
			if err != nil {
				return err
			}
		} else {
			pid, err = addGrantIdentity(client, cmdOptions.Identity)
			if err != nil {
				return err
			}
		}
	}

	_, err = client.CreateGrant(&api.CreateGrantRequest{
		Subject:   pid,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Destination,
	})
	if err != nil {
		return err
	}

	cli.Output("Access granted!")

	return nil
}

// getSubjectPolymorphicID gets the ID for either the group or identity in the subject of a grant
func getSubjectPolymorphicID(client *api.Client, subject string, isGroup bool) (uid.PolymorphicID, error) {
	if isGroup {
		identity, err := GetGroupByName(client, subject)
		if err != nil {
			return "", err
		}
		return uid.NewGroupPolymorphicID(identity.ID), nil
	}

	identity, err := GetIdentityByName(client, subject)
	if err != nil {
		return "", err
	}
	return uid.NewIdentityPolymorphicID(identity.ID), nil
}

func GetGroupByName(client *api.Client, name string) (*api.Group, error) {
	groups, err := client.ListGroups(api.ListGroupsRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if groups.Count == 0 {
		return nil, ErrIdentityNotFound
	}

	if groups.Count != 1 {
		return nil, fmt.Errorf("invalid groups response, there should only be one group that matches a name, but multiple were found")
	}

	return &groups.Items[0], nil
}

func subjectNameFromGrant(client *api.Client, g api.Grant) (name string, err error) {
	id, err := g.Subject.ID()
	if err != nil {
		return "", err
	}

	if g.Subject.IsIdentity() {
		identity, err := client.GetIdentity(id)
		if err != nil {
			return "", err
		}

		return identity.Name, nil
	}

	if g.Subject.IsGroup() {
		group, err := client.GetGroup(id)
		if err != nil {
			return "", err
		}

		return group.Name, nil
	}

	return "", fmt.Errorf("unrecognized grant subject")
}

func addGrantIdentity(client *api.Client, name string) (uid.PolymorphicID, error) {
	created, err := CreateIdentity(&api.CreateIdentityRequest{Name: name})
	if err != nil {
		return "", err
	}
	fmt.Printf("New unlinked identity %q added to Infra\n", name)

	return uid.NewIdentityPolymorphicID(created.ID), nil
}

func addGrantGroup(client *api.Client, name string) (uid.PolymorphicID, error) {
	created, err := client.CreateGroup(&api.CreateGroupRequest{Name: name})
	if err != nil {
		return "", err
	}
	fmt.Printf("New group %q added to Infra\n", name)
	return uid.NewGroupPolymorphicID(created.ID), nil
}
