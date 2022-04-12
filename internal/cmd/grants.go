package cmd

import (
	"fmt"
	"net/mail"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type grantsCmdOptions struct {
	Identity    string `mapstructure:"identity"`
	Destination string `mapstructure:"destination"`
	IsGroup     bool   `mapstructure:"group"`
	Role        string `mapstructure:"role"`
}

func newGrantsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "grants",
		Short:   "Manage access to destinations",
		Aliases: []string{"grant"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newGrantsListCmd())
	cmd.AddCommand(newGrantAddCmd())
	cmd.AddCommand(newGrantRemoveCmd())

	return cmd
}

func newGrantsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List grants",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options grantsCmdOptions
			if err := parseOptions(cmd, &options, "INFRA_GRANTS"); err != nil {
				return err
			}

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
			for _, g := range grants {
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
				printTable(rows)
			} else {
				fmt.Println("No grants found")
			}

			return nil
		},
	}

	cmd.Flags().String("destination", "", "Filter by destination")
	return cmd
}

func newGrantRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove IDENTITY DESTINATION",
		Aliases: []string{"rm"},
		Short:   "Revoke access to a destination",
		Long: `Revokes access that user has to the destination.

IDENTITY is one that was being given access.
DESTINATION is what the identity will lose access to. 

Use [--role] to specify the exact grant being deleted. 
If not specified, it will revoke all roles for that user within the destination. 

Use [--group] or [-g] if identity is of type group. 
$ infra grants remove devGroup -g ...
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options grantsCmdOptions
			if err := parseOptions(cmd, &options, "INFRA_GRANTS"); err != nil {
				return err
			}

			options.Identity = args[0]
			options.Destination = args[1]

			return removeGrant(options)
		},
	}

	cmd.Flags().BoolP("group", "g", false, "Group to revoke access from")
	cmd.Flags().String("role", "", "Role to revoke")

	return cmd
}

func removeGrant(cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	identityType, err := getIdentityType(cmdOptions.Identity, cmdOptions.IsGroup)
	if err != nil {
		return err
	}

	id, err := getIDByName(client, cmdOptions.Identity, identityType)
	if err != nil {
		return err
	}

	grants, err := client.ListGrants(api.ListGrantsRequest{
		Subject:   id,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Destination,
	})
	if err != nil {
		return err
	}

	for _, g := range grants {
		err := client.DeleteGrant(g.ID)
		if err != nil {
			return err
		}
	}

	fmt.Println("Access revoked!")

	return nil
}

func newGrantAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add IDENTITY DESTINATION",
		Short: "Grant access to a destination",
		Long: `Grant one or more identities access to a destination. 

IDENTITY is the subject that is being given access.
DESTINATION is what the identity will gain access to. 

Use [--role] if further fine grained permissions are needed. If not specified, user will gain the permission 'connect' to the destination. 
$ infra grants add ... -role admin ...

Use [--group] or [-g] if identity is of type group. 
$ infra grants add devGroup -group ...
$ infra grants add devGroup -g ...

For full documentation on grants with more examples, see: 
  https://github.com/infrahq/infra/blob/main/docs/guides
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options grantsCmdOptions
			if err := parseOptions(cmd, &options, "INFRA_GRANTS"); err != nil {
				return err
			}

			options.Identity = args[0]
			options.Destination = args[1]

			return addGrant(options)
		},
	}

	cmd.Flags().BoolP("group", "g", false, "Required if identity is of type 'group'")
	cmd.Flags().String("role", models.BasePermissionConnect, "Type of access that identity will be given")
	return cmd
}

func addGrant(cmdOptions grantsCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	identityType, err := getIdentityType(cmdOptions.Identity, cmdOptions.IsGroup)
	if err != nil {
		return err
	}

	id, err := getIDByName(client, cmdOptions.Identity, identityType)
	if err != nil {
		return err
	}

	_, err = client.CreateGrant(&api.CreateGrantRequest{
		Subject:   id,
		Privilege: cmdOptions.Role,
		Resource:  cmdOptions.Destination,
	})
	if err != nil {
		return err
	}

	fmt.Println("Access granted!")

	return nil
}

type identityType int8

const (
	userType identityType = iota
	machineType
	groupType
)

// Unless explicitly specified as a group, identity will be a user if email, machine if not.
func getIdentityType(s string, isGroup bool) (identityType, error) {
	if isGroup {
		return groupType, nil
	}

	maybeName := regexp.MustCompile("^[a-zA-Z0-9-_./]+$")
	if maybeName.MatchString(s) {
		nameMinLength := 1
		nameMaxLength := 256

		if len(s) < nameMinLength {
			return machineType, fmt.Errorf("invalid name: does not meet minimum length requirement of %d characters", nameMinLength)
		}

		if len(s) > nameMaxLength {
			return machineType, fmt.Errorf("invalid name: exceed maximum length requirement of %d characters", nameMaxLength)
		}

		return machineType, nil
	}

	_, err := mail.ParseAddress(s)
	if err != nil {
		return userType, fmt.Errorf("invalid email: %q", s)
	}

	return userType, nil
}

func getIDByName(client *api.Client, name string, identityType identityType) (uid.PolymorphicID, error) {
	var id uid.PolymorphicID
	switch identityType {
	case groupType:
		groups, err := client.ListGroups(api.ListGroupsRequest{Name: name})
		if err != nil {
			return "", err
		}

		switch len(groups) {
		case 0:
			created, err := client.CreateGroup(&api.CreateGroupRequest{Name: name})
			if err != nil {
				return "", err
			}
			fmt.Printf("New group %q added to Infra\n", name)

			id = uid.NewGroupPolymorphicID(created.ID)
		case 1:
			id = uid.NewGroupPolymorphicID(groups[0].ID)
		default:
			panic(fmt.Sprintf(DuplicateEntryPanic, "group", name))
		}
	case userType, machineType:
		identities, err := client.ListIdentities(api.ListIdentitiesRequest{Name: name})
		if err != nil {
			return "", err
		}

		switch len(identities) {
		case 0:
			created, err := CreateIdentity(name)
			if err != nil {
				return "", err
			}
			fmt.Printf("New unlinked identity %q added to Infra\n", name)

			id = uid.NewIdentityPolymorphicID(created.ID)
		case 1:
			id = uid.NewIdentityPolymorphicID(identities[0].ID)
		default:
			panic(fmt.Sprintf(DuplicateEntryPanic, "identity", name))
		}
	default:
		panic("identity must be either user, machine, or group")
	}

	return id, nil
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
