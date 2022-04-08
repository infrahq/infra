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

type grantsCmdOptionsNew struct {
	Identity    string `mapstructure:"identity"`
	Destination string `mapstructure:"destination"`
	IsGroup     bool   `mapstructure:"group"`
	Role        string `mapstructure:"role"`
}

func newGrantAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add IDENTITY DESTINATION",
		Short: "Grant access to a destination",
		Long: `Grant one or more identities access to a destination. 

IDENTITY is one that is being given access.
DESTINATION is what the identity will gain access to. 

Use [--role] if further fine grained permissions are needed. If not specified, user will gain the permission 'connect' to the destination. 
$ infra grants add ... -role admin ...

Use [--group] or [-g] if identity is of type group. 
$ infra grants add devGroup -group ...
$ infra grants add devGroup -g ...

For full documentation on grants, see  https://github.com/infrahq/infra/blob/main/docs/using-infra/grants.md 
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options grantsCmdOptionsNew
			if err := parseOptions(cmd, &options, "INFRA_ACCESS"); err != nil {
				return err
			}

			options.Identity = args[0]
			options.Destination = args[1]

			return addGrant(options)
		},
	}

	cmd.Flags().BoolP("group", "g", false, "Marks identity as type 'group'")
	cmd.Flags().String("role", models.BasePermissionConnect, "Type of access that identity will be given")
	return cmd
}

func addGrant(cmdOptions grantsCmdOptionsNew) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	identityType, err := getIdentityType(cmdOptions.Identity, cmdOptions.IsGroup)
	if err != nil {
		return err
	}

	var id uid.PolymorphicID
	switch identityType {
	case groupType:
		groups, err := client.ListGroups(api.ListGroupsRequest{Name: cmdOptions.Identity})
		if err != nil {
			return err
		}

		switch len(groups) {
		case 0:
			return fmt.Errorf("No group of name %s exists", cmdOptions.Identity)
		case 1:
			id = uid.NewGroupPolymorphicID(groups[0].ID)
		case 2:
			panic(fmt.Sprintf(DuplicateEntryPanic, "group", cmdOptions.Identity))
		}
	case userType, machineType:
		identities, err := client.ListIdentities(api.ListIdentitiesRequest{Name: cmdOptions.Identity})
		if err != nil {
			return err
		}

		switch len(identities) {
		case 0:
			return fmt.Errorf("No identity of name %s exists", cmdOptions.Identity)
		case 1:
			id = uid.NewIdentityPolymorphicID(identities[0].ID)
		case 2:
			panic(fmt.Sprintf(DuplicateEntryPanic, "identity", cmdOptions.Identity))
		}
	default:
		panic("kind must be either user, machine, or group")
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
