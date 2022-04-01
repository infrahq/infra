package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type grantsCmdOptions struct {
	User     string `mapstructure:"user"`
	Group    string `mapstructure:"group"`
	Machine  string `mapstructure:"machine"`
	Provider string `mapstructure:"provider"`
	Role     string `mapstructure:"role"`
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
			var options grantsCmdOptions
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
				options.Role = models.BasePermissionConnect
			}

			var id uid.PolymorphicID

			if options.User != "" {
				// create user if they don't exist
				users, err := client.ListIdentities(api.ListIdentitiesRequest{Name: options.User, ProviderID: provider.ID})
				if err != nil {
					return err
				}

				if len(users) == 0 {
					newUser, err := client.CreateIdentity(&api.CreateIdentityRequest{
						Name:       options.User,
						Kind:       models.UserKind.String(),
						ProviderID: provider.ID,
					})
					if err != nil {
						return err
					}

					id = uid.NewIdentityPolymorphicID(newUser.ID)
				} else {
					id = uid.NewIdentityPolymorphicID(users[0].ID)
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
				machines, err := client.ListIdentities(api.ListIdentitiesRequest{Name: options.Machine})
				if err != nil {
					return err
				}

				if len(machines) == 0 {
					infraProvider, err := GetProviderByName(client, models.InternalInfraProviderName)
					if err != nil {
						logging.S.Debug(err)
						return fmt.Errorf("no infra provider found, to manage local users create a local provider named 'infra'")
					}

					newMachine, err := client.CreateIdentity(&api.CreateIdentityRequest{
						Name:       options.Machine,
						Kind:       models.MachineKind.String(),
						ProviderID: infraProvider.ID,
					})
					if err != nil {
						return err
					}

					id = uid.NewIdentityPolymorphicID(newMachine.ID)
				} else {
					id = uid.NewIdentityPolymorphicID(machines[0].ID)
				}
			}

			_, err = client.CreateGrant(&api.CreateGrantRequest{
				Subject:   id,
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
			var options grantsCmdOptions
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
				users, err := client.ListIdentities(api.ListIdentitiesRequest{Name: options.User, ProviderID: provider.ID})
				if err != nil {
					return err
				}

				if len(users) == 0 {
					return errors.New("no such user")
				}

				id = uid.NewIdentityPolymorphicID(users[0].ID)
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
				machines, err := client.ListIdentities(api.ListIdentitiesRequest{Name: options.Machine})
				if err != nil {
					return err
				}

				if len(machines) == 0 {
					return errors.New("no such machine")
				}

				id = uid.NewIdentityPolymorphicID(machines[0].ID)
			}

			grants, err := client.ListGrants(api.ListGrantsRequest{
				Subject:   id,
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
