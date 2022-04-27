package cmd

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"

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

func newGrantsCmd() *cobra.Command {
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

	cmd.AddCommand(newGrantsListCmd())
	cmd.AddCommand(newGrantAddCmd())
	cmd.AddCommand(newGrantRemoveCmd())

	return cmd
}

func newGrantsListCmd() *cobra.Command {
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
				printTable(rows, TODO)
			} else {
				fmt.Println("No grants found")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&options.Destination, "destination", "", "Filter by destination")
	return cmd
}

func newGrantRemoveCmd() *cobra.Command {
	var options grantsCmdOptions

	cmd := &cobra.Command{
		Use:     "remove IDENTITY DESTINATION",
		Aliases: []string{"rm"},
		Short:   "Revoke an identity's access from a destination",
		Example: `# Remove all grants of an identity in a destination
$ infra grants remove janedoe@example.com kubernetes.docker-desktop 
$ infra grants remove machine-a kubernetes.docker-desktop

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
			return removeGrant(options)
		},
	}

	cmd.Flags().BoolVarP(&options.IsGroup, "group", "g", false, "Group to revoke access from")
	cmd.Flags().StringVar(&options.Role, "role", "", "Role to revoke")
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
	var options grantsCmdOptions

	cmd := &cobra.Command{
		Use:   "add IDENTITY DESTINATION",
		Short: "Grant an identity access to a destination",
		Example: `# Grant an identity access to a destination
$ infra grants add johndoe@example.com kubernetes.docker-desktop 
$ infra grants add machine-a kubernetes.docker-desktop

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
			return addGrant(options)
		},
	}

	cmd.Flags().BoolVarP(&options.IsGroup, "group", "g", false, "Required if identity is of type 'group'")
	cmd.Flags().StringVar(&options.Role, "role", models.BasePermissionConnect, "Type of access that identity will be given")
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
		if !errors.Is(err, ErrIdentityNotFound) {
			return err
		}
		id, err = addGrantIdentity(client, cmdOptions.Identity, identityType)
		if err != nil {
			return err
		}
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
			return "", ErrIdentityNotFound
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
			return "", ErrIdentityNotFound
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

func addGrantIdentity(client *api.Client, name string, identityType identityType) (uid.PolymorphicID, error) {
	var id uid.PolymorphicID
	switch identityType {
	case groupType:
		created, err := client.CreateGroup(&api.CreateGroupRequest{Name: name})
		if err != nil {
			return "", err
		}
		fmt.Printf("New group %q added to Infra\n", name)
		id = uid.NewGroupPolymorphicID(created.ID)
	case userType, machineType:
		created, err := CreateIdentity(&api.CreateIdentityRequest{Name: name})
		if err != nil {
			return "", err
		}
		fmt.Printf("New unlinked identity %q added to Infra\n", name)

		id = uid.NewIdentityPolymorphicID(created.ID)
	default:
		panic("identity must be either user, machine, or group")
	}

	return id, nil
}
