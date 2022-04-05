package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
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
