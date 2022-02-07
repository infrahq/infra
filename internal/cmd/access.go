package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/uid"
)

func newAccessListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List access",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}
	
			grants, err := client.ListGrants(api.ListGrantsRequest{})
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
	var (
		user     string
		group    string
		provider string
		role     string
	)

	cmd := &cobra.Command{
		Use:   "grant DESTINATION",
		Short: "Grant access",
		Example: `
# Grant user admin access to a cluster
infra grant -u suzie@acme.com -r admin kubernetes.production 

# Grant group admin access to a namespace
infra grant -g Engineering -r admin kubernetes.production.default

# Grant user admin access to infra itself
infra grant -u admin@acme.com -r admin infra
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			providers, err := client.ListProviders(provider)
			if err != nil {
				return err
			}

			if len(providers) == 0 {
				return errors.New("No identity providers connected")
			}

			if len(providers) > 1 {
				return errors.New("Specify provider with -p or --provider")
			}

			if group != "" {
				if user != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				// create user if they don't exist
				groups, err := client.ListGroups(api.ListGroupsRequest{Name: group})
				if err != nil {
					return err
				}

				var id uid.ID

				if len(groups) == 0 {
					newGroup, err := client.CreateGroup(&api.CreateGroupRequest{
						Name:       group,
						ProviderID: providers[0].ID,
					})
					if err != nil {
						return err
					}

					id = newGroup.ID
				} else {
					id = groups[0].ID
				}

				_, err = client.CreateGrant(&api.CreateGrantRequest{
					Identity:  fmt.Sprintf("g:%s", id),
					Resource:  args[0],
					Privilege: role,
				})
				if err != nil {
					return err
				}
			}

			if user != "" {
				if group != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				// create user if they don't exist
				users, err := client.ListUsers(api.ListUsersRequest{Email: user})
				if err != nil {
					return err
				}

				var id uid.ID

				if len(users) == 0 {
					newUser, err := client.CreateUser(&api.CreateUserRequest{
						Email:      user,
						ProviderID: providers[0].ID,
					})
					if err != nil {
						return err
					}

					id = newUser.ID
				} else {
					id = users[0].ID
				}

				_, err = client.CreateGrant(&api.CreateGrantRequest{
					Identity:  fmt.Sprintf("u:%s", id),
					Resource:  args[0],
					Privilege: role,
				})

				if err != nil {
					return err
				}
			}

			fmt.Println("Access granted")

			return nil
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "User to grant access to")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Group to grant access to")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider from which to grant user access to")
	cmd.Flags().StringVarP(&role, "role", "r", "", "Role to grant")

	return cmd
}

func newAccessRevokeCmd() *cobra.Command {
	var (
		user     string
		group    string
		provider string
		role     string
	)

	cmd := &cobra.Command{
		Use:   "revoke DESTINATION",
		Short: "Revoke access",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			providers, err := client.ListProviders(provider)
			if err != nil {
				return err
			}

			if len(providers) == 0 {
				return errors.New("No identity providers connected")
			}

			if len(providers) > 1 {
				return errors.New("Specify provider with -p or --provider")
			}

			var identity string

			if group != "" {
				if user != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				groups, err := client.ListGroups(api.ListGroupsRequest{Name: group})
				if err != nil {
					return err
				}

				if len(groups) == 0 {
					return errors.New("no such group")
				}

				identity = "g:" + groups[0].ID.String()
			}

			if user != "" {
				if group != "" {
					return errors.New("only one of -g and -u are allowed")
				}

				users, err := client.ListUsers(api.ListUsersRequest{Email: user})
				if err != nil {
					return err
				}

				if len(users) == 0 {
					return errors.New("no such user")
				}

				identity = "u:" + users[0].ID.String()
			}

			grants, err := client.ListGrants(api.ListGrantsRequest{Resource: args[0], Identity: identity, Privilege: role})
			if err != nil {
				return err
			}

			for _, g := range grants {
				err := client.DeleteGrant(g.ID)
				if err != nil {
					return err
				}
			}

			fmt.Println("Access revoked")

			return nil
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "User to revoke access from")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Group to revoke access from")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider from which to revoke access from")
	cmd.Flags().StringVarP(&role, "role", "r", "", "Role to revoke")

	return cmd
}
