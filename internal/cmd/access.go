package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/api"
)

type accessOptions struct {
	User     string `mapstructure:"user"`
	Group    string `mapstructure:"group"`
	Provider string `mapstructure:"provider"`
	Role     string `mapstructure:"role"`
}

func newAccessListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [DESTINATION]",
		Aliases: []string{"ls"},
		Short:   "List access",
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

				provider, identity, err := info(client, g)
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

func newAccessGrantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant DESTINATION",
		Short: "Grant access",
		Example: `
# Grant user admin access to a cluster
infra access grant -u suzie@acme.com -r admin kubernetes.production

# Grant group admin access to a namespace
infra access grant -g Engineering -r admin kubernetes.production.default

# Grant user admin access to infra itself
infra access grant -u admin@acme.com -r admin infra
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

			providers, err := client.ListProviders(options.Provider)
			if err != nil {
				return err
			}

			if len(providers) == 0 {
				return errors.New("no identity providers connected")
			}

			if len(providers) > 1 {
				return errors.New("specify provider with -p or --provider")
			}

			if options.User != "" && options.Group != "" {
				return errors.New("only allowed one of --user or --group")
			}

			if options.Role == "" {
				return errors.New("specific role with -r or --role")
			}

			var id strings.Builder

			if options.User != "" {
				// create user if they don't exist
				users, err := client.ListUsers(api.ListUsersRequest{Email: options.User})
				if err != nil {
					return err
				}

				id.WriteString("u:")

				if len(users) == 0 {
					newUser, err := client.CreateUser(&api.CreateUserRequest{
						Email:      options.User,
						ProviderID: providers[0].ID,
					})
					if err != nil {
						return err
					}

					id.WriteString(newUser.ID.String())
				} else {
					id.WriteString(users[0].ID.String())
				}
			}

			if options.Group != "" {
				// create group if they don't exist
				groups, err := client.ListGroups(api.ListGroupsRequest{Name: options.Group})
				if err != nil {
					return err
				}

				id.WriteString("g:")

				if len(groups) == 0 {
					newGroup, err := client.CreateGroup(&api.CreateGroupRequest{
						Name:       options.Group,
						ProviderID: providers[0].ID,
					})
					if err != nil {
						return err
					}

					id.WriteString(newGroup.ID.String())
				} else {
					id.WriteString(groups[0].ID.String())
				}
			}

			_, err = client.CreateGrant(&api.CreateGrantRequest{
				Identity:  id.String(),
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
	cmd.Flags().StringP("group", "g", "", "Group to grant access to")
	cmd.Flags().StringP("provider", "p", "", "Provider from which to grant user access to")
	cmd.Flags().StringP("role", "r", "", "Role to grant")

	return cmd
}

func newAccessRevokeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke DESTINATION",
		Short: "Revoke access",
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

			providers, err := client.ListProviders(options.Provider)
			if err != nil {
				return err
			}

			if len(providers) == 0 {
				return errors.New("No identity providers connected")
			}

			if len(providers) > 1 {
				return errors.New("Specify provider with -p or --provider")
			}

			if options.User != "" && options.Group != "" {
				return errors.New("only allowed one of --user or --group")
			}

			var id string

			if options.User != "" {
				users, err := client.ListUsers(api.ListUsersRequest{Email: options.User})
				if err != nil {
					return err
				}

				if len(users) == 0 {
					return errors.New("no such user")
				}

				id = fmt.Sprintf("u:%s", users[0].ID.String())
			}

			if options.Group != "" {
				groups, err := client.ListGroups(api.ListGroupsRequest{Name: options.Group})
				if err != nil {
					return err
				}

				if len(groups) == 0 {
					return errors.New("no such group")
				}

				id = fmt.Sprintf("g:%s", groups[0].ID.String())
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
	cmd.Flags().StringP("provider", "p", "", "Provider from which to revoke access from")
	cmd.Flags().StringP("role", "r", "", "Role to revoke")

	return cmd
}
