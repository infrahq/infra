package cmd

import (
	"errors"
	"fmt"
	"net/mail"
	"os"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type identityOptions struct {
	Description string `mapstructure:"description"`
	Password    bool   `mapstructure:"password"`
}

func newIdentitiesAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Create an identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			var options identityOptions
			if err := parseOptions(cmd, &options, "INFRA_IDENTITY"); err != nil {
				return err
			}

			if isUser(name) {
				userCreateResp, err := createUser(name)
				if err != nil {
					return err
				}
				fmt.Println("user identity created")
				fmt.Printf("one time password: %s \n", userCreateResp.OneTimePassword)
			} else {
				err := createMachine(name, &options)
				if err != nil {
					return err
				}
				fmt.Println("machine identity created")
			}

			return nil
		},
	}

	cmd.Flags().StringP("description", "d", "", "Description of a machine identity")

	return cmd
}

func newIdentitiesEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit NAME",
		Short: "Update an identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			var options identityOptions
			if err := parseOptions(cmd, &options, "INFRA_IDENTITY"); err != nil {
				return err
			}

			if isUser(name) {
				if !options.Password {
					return errors.New("specify the --password flag to update the password")
				}

				newPassword := ""
				err := survey.AskOne(&survey.Password{Message: "New Password:"}, &newPassword, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
				if err != nil {
					return err
				}

				err = updateUser(name, newPassword)
				if err != nil {
					return err
				}

				fmt.Println("user identity updated")
			} else {
				fmt.Println("machine identities have no editable fields")
			}

			return nil
		},
	}

	cmd.Flags().BoolP("password", "p", false, "Prompt to update a local user's password")

	return cmd
}

func newIdentitiesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all identities",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			type row struct {
				Name        string `header:"Name"`
				Type        string `header:"Type"`
				Provider    string `header:"Provider"`
				Description string `header:"Description"`
			}

			machines, err := client.ListMachines(api.ListMachinesRequest{})
			if err != nil {
				return err
			}

			var rows []row

			for _, m := range machines {
				rows = append(rows, row{
					Name:        m.Name,
					Type:        "machine",
					Description: m.Description,
				})
			}

			users, err := client.ListUsers(api.ListUsersRequest{})
			if err != nil {
				return err
			}

			providers := make(map[uid.ID]string)

			for _, u := range users {
				if providers[u.ProviderID] == "" {
					p, err := client.GetProvider(u.ProviderID)
					if err != nil {
						logging.S.Debugf("unable to retrieve user provider: %w", err)
					} else {
						providers[p.ID] = p.Name
					}
				}

				rows = append(rows, row{
					Name:     u.Email,
					Type:     "user",
					Provider: providers[u.ProviderID],
				})
			}

			printTable(rows)

			return nil
		},
	}
}

func newIdentitiesRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove NAME",
		Short: "Delete an identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			var options identityOptions
			if err := parseOptions(cmd, &options, "INFRA_IDENTITY"); err != nil {
				return err
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			if isUser(name) {
				users, err := client.ListUsers(api.ListUsersRequest{Email: name})
				if err != nil {
					return err
				}

				for _, u := range users {
					err := client.DeleteUser(u.ID)
					if err != nil {
						return err
					}
				}
			} else {
				machines, err := client.ListMachines(api.ListMachinesRequest{Name: name})
				if err != nil {
					return err
				}

				for _, m := range machines {
					err := client.DeleteMachine(m.ID)
					if err != nil {
						return err
					}
				}
			}

			return nil
		},
	}

	return cmd
}

func isUser(name string) bool {
	// infer based on the name being an email
	_, err := mail.ParseAddress(name)
	if err != nil {
		logging.S.Debug(err)
		return false
	}
	return true
}

func createMachine(name string, options *identityOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	_, err = client.CreateMachine(&api.CreateMachineRequest{Name: name, Description: options.Description})
	if err != nil {
		return err
	}

	return nil
}

func createUser(email string) (*api.CreateUserResponse, error) {
	client, err := defaultAPIClient()
	if err != nil {
		return nil, err
	}

	infraProvider, err := GetProviderByName(client, models.InternalInfraProviderName)
	if err != nil {
		logging.S.Debug(err)
		return nil, fmt.Errorf("no infra provider found, to manage local user create a local provider named 'infra'")
	}

	resp, err := client.CreateUser(&api.CreateUserRequest{Email: email, ProviderID: infraProvider.ID})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func updateUser(name, newPassword string) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	infraProvider, err := GetProviderByName(client, models.InternalInfraProviderName)
	if err != nil {
		logging.S.Debug(err)
		return fmt.Errorf("no infra provider found, to manage local users create a local provider named 'infra'")
	}

	user := &api.User{}

	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	if config.ProviderID == infraProvider.ID && config.Name == name {
		// this is a user updating their own password
		currentID, err := config.PolymorphicID.ID()
		if err != nil {
			return err
		}
		user.ID = currentID
	} else {
		user, err = getUserFromName(client, name, infraProvider)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return fmt.Errorf("the user being updated must exist in the local infra identity provider: %w", err)
			}
			return err
		}
		fmt.Println("setting one time password")
	}

	if _, err := client.UpdateUser(&api.UpdateUserRequest{ID: user.ID, Password: newPassword}); err != nil {
		return err
	}

	return nil
}

func getMachineFromName(client *api.Client, name string) (*api.Machine, error) {
	machines, err := client.ListMachines(api.ListMachinesRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if len(machines) == 0 {
		return nil, fmt.Errorf("no machine found with this name")
	}

	if len(machines) != 1 {
		return nil, fmt.Errorf("invalid machines response, there should only be one machine that matches a name, but multiple were found")
	}

	return &machines[0], nil
}

func getUserFromName(client *api.Client, name string, provider *api.Provider) (*api.User, error) {
	users, err := client.ListUsers(api.ListUsersRequest{Email: name, ProviderID: provider.ID})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	if len(users) != 1 {
		return nil, fmt.Errorf("invalid users response, there should only be one user that matches a name, but multiple were found")
	}

	return &users[0], nil
}
