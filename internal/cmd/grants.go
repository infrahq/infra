package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type accessOptions struct {
	User     string `mapstructure:"user"`
	Group    string `mapstructure:"group"`
	Machine  string `mapstructure:"machine"`
	Provider string `mapstructure:"provider"`
	Role     string `mapstructure:"role"`
}

func newGrantsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [DESTINATION]",
		Aliases: []string{"ls"},
		Short:   "List grants",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resource string
			if len(args) > 0 {
				resource = args[0]
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			grants, err := client.ListGrants(api.ListGrantsRequest{Resource: resource})
			if err != nil {
				return err
			}

			type row struct {
				Provider string `header:"PROVIDER"`
				Identity string `header:"IDENTITY"`
				Access   string `header:"ACCESS"`
				Resource string `header:"RESOURCE"`
			}

			var rows []row
			for _, g := range grants {
				if strings.HasPrefix(g.Resource, "infra") {
					continue
				}

				provider, identity, err := listInfo(client, g)
				if err != nil {
					return err
				}

				rows = append(rows, row{
					Provider: provider,
					Identity: identity,
					Access:   g.Privilege,
					Resource: g.Resource,
				})
			}

			printTable(rows)

			return nil
		},
	}
}

func newGrantAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add DESTINATION",
		Short: "Grant access to a destination",
		Example: `
# Grant user admin access to a cluster
$ infra grants add -u suzie@acme.com -r admin kubernetes.production

# Grant group admin access to a namespace
$ infra grants add -g Engineering -r admin kubernetes.production.default

# Grant user admin access to infra itself
$ infra grants add -u admin@acme.com -r admin infra
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options accessOptions
			if err := parseOptions(cmd, &options, "INFRA_ACCESS"); err != nil {
				return err
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			var provider *api.Provider

			if options.Machine == "" {
				provider, err = GetProviderByName(client, options.Provider)
				if err != nil {
					if errors.Is(err, ErrProviderNotUnique) {
						return fmt.Errorf("specify provider with -p or --provider: %w", err)
					}
					return err
				}

				if options.User != "" && options.Group != "" {
					return errors.New("only allowed one of --user or --group")
				}
			} else if options.User != "" || options.Group != "" {
				return errors.New("cannot specify --user or --group with --machine")
			}

			if options.Role == "" {
				return errors.New("specify role with -r or --role")
			}

			var id uid.PolymorphicID

			if options.User != "" {
				// create user if they don't exist
				users, err := client.ListUsers(api.ListUsersRequest{Email: options.User, ProviderID: provider.ID})
				if err != nil {
					return err
				}

				if len(users) == 0 {
					newUser, err := client.CreateUser(&api.CreateUserRequest{
						Email:      options.User,
						ProviderID: provider.ID,
					})
					if err != nil {
						return err
					}

					id = uid.NewUserPolymorphicID(newUser.ID)
				} else {
					id = uid.NewUserPolymorphicID(users[0].ID)
				}
			}

			if options.Group != "" {
				// create group if they don't exist
				groups, err := client.ListGroups(api.ListGroupsRequest{Name: options.Group, ProviderID: provider.ID})
				if err != nil {
					return err
				}

				if len(groups) == 0 {
					newGroup, err := client.CreateGroup(&api.CreateGroupRequest{
						Name:       options.Group,
						ProviderID: provider.ID,
					})
					if err != nil {
						return err
					}

					id = uid.NewGroupPolymorphicID(newGroup.ID)
				} else {
					id = uid.NewGroupPolymorphicID(groups[0].ID)
				}
			}

			if options.Machine != "" {
				// create machine if they don't exist
				machines, err := client.ListMachines(api.ListMachinesRequest{Name: options.Machine})
				if err != nil {
					return err
				}

				if len(machines) == 0 {
					newMachine, err := client.CreateMachine(&api.CreateMachineRequest{
						Name: options.Machine,
					})
					if err != nil {
						return err
					}

					id = uid.NewMachinePolymorphicID(newMachine.ID)
				} else {
					id = uid.NewMachinePolymorphicID(machines[0].ID)
				}
			}

			_, err = client.CreateGrant(&api.CreateGrantRequest{
				Identity:  id,
				Privilege: options.Role,
				Resource:  args[0],
			})
			if err != nil {
				return err
			}

			fmt.Println("Access granted!")

			return nil
		},
	}

	cmd.Flags().StringP("user", "u", "", "User to grant access to")
	cmd.Flags().StringP("machine", "m", "", "Machine to grant access to")
	cmd.Flags().StringP("group", "g", "", "Group to grant access to")
	cmd.Flags().StringP("provider", "p", "", "Provider from which to grant user access to")
	cmd.Flags().StringP("role", "r", "", "Role to grant")

	return cmd
}

func newGrantRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove DESTINATION",
		Short: "Revoke access to a destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options accessOptions
			if err := parseOptions(cmd, &options, "INFRA_ACCESS"); err != nil {
				return err
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			var provider *api.Provider

			if options.Machine == "" {
				provider, err = GetProviderByName(client, options.Provider)
				if err != nil {
					if errors.Is(err, ErrProviderNotUnique) {
						return fmt.Errorf("specify provider with -p or --provider: %w", err)
					}
					return err
				}

				if options.User != "" && options.Group != "" {
					return errors.New("only allowed one of --user or --group")
				}
			} else if options.User != "" || options.Group != "" {
				return errors.New("cannot specify --user or --group with --machine")
			}

			var id uid.PolymorphicID

			if options.User != "" {
				users, err := client.ListUsers(api.ListUsersRequest{Email: options.User, ProviderID: provider.ID})
				if err != nil {
					return err
				}

				if len(users) == 0 {
					return errors.New("no such user")
				}

				id = uid.NewUserPolymorphicID(users[0].ID)
			}

			if options.Group != "" {
				groups, err := client.ListGroups(api.ListGroupsRequest{Name: options.Group, ProviderID: provider.ID})
				if err != nil {
					return err
				}

				if len(groups) == 0 {
					return errors.New("no such group")
				}

				id = uid.NewGroupPolymorphicID(groups[0].ID)
			}

			if options.Machine != "" {
				machines, err := client.ListMachines(api.ListMachinesRequest{Name: options.Machine})
				if err != nil {
					return err
				}

				if len(machines) == 0 {
					return errors.New("no such machine")
				}

				id = uid.NewMachinePolymorphicID(machines[0].ID)
			}

			grants, err := client.ListGrants(api.ListGrantsRequest{
				Identity:  id,
				Privilege: options.Role,
				Resource:  args[0],
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
		},
	}

	cmd.Flags().StringP("user", "u", "", "User to revoke access from")
	cmd.Flags().StringP("group", "g", "", "Group to revoke access from")
	cmd.Flags().StringP("machine", "m", "", "Machine to revoke access from")
	cmd.Flags().StringP("provider", "p", "", "Provider from which to revoke access from")
	cmd.Flags().StringP("role", "r", "", "Role to revoke")

	return cmd
}
