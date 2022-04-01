package cmd

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"regexp"

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
		Use:   "add NAME|EMAIL",
		Short: "Create an identity.",
		Long: `Create a machine identity with NAME or a user identity with EMAIL.

NAME must only contain alphanumeric characters ('a-z', 'A-Z', '0-9') or the
special characters '-', '_', or '/' and has a maximum length of 256 characters.

EMAIL must contain a valid email address in the form of "<local>@<domain>".
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrEmail := args[0]

			var options identityOptions
			if err := parseOptions(cmd, &options, ""); err != nil {
				return err
			}

			name, email, err := checkNameOrEmail(nameOrEmail)
			if err != nil {
				return err
			}

			if name != "" {
				err := createMachine(name, &options)
				if err != nil {
					return err
				}

				fmt.Fprintf(os.Stderr, "Created machine identity.\n")
				fmt.Printf("Name: %s\n", name)
			}

			if email != "" {
				userCreateResp, err := createUser(email)
				if err != nil {
					return err
				}

				fmt.Fprintf(os.Stderr, "Created user identity.\n")
				fmt.Printf("Username: %s\n", email)
				fmt.Printf("Password: %s\n", userCreateResp.OneTimePassword)
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
			nameOrEmail := args[0]

			var options identityOptions
			if err := parseOptions(cmd, &options, ""); err != nil {
				return err
			}

			name, email, err := checkNameOrEmail(nameOrEmail)
			if err != nil {
				return err
			}

			if name != "" {
				fmt.Println("machine identities have no editable fields")
			}

			if email != "" {
				if !options.Password {
					return errors.New("specify the --password flag to update the password")
				}

				newPassword, err := promptUpdatePassword("")
				if err != nil {
					return err
				}

				if err = updateUser(name, newPassword); err != nil {
					return err
				}
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
						logging.S.Debugf("unable to retrieve user provider: %s", err)
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
			nameOrEmail := args[0]

			var options identityOptions
			if err := parseOptions(cmd, &options, ""); err != nil {
				return err
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			name, email, err := checkNameOrEmail(nameOrEmail)
			if err != nil {
				return err
			}

			if name != "" {
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

			if email != "" {
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
			}

			return nil
		},
	}

	return cmd
}

// checkNameOrEmail infers whether the input s specifies a user identity (email) or a machine
// identity (name). The input is considered a name if it has the format `^[a-zA-Z0-9-_/]+$`.
// All other input formats will be considered an email. If an email input fails validation,
// return an error.
func checkNameOrEmail(s string) (string, string, error) {
	maybeName := regexp.MustCompile("^[a-zA-Z0-9-_/]+$")
	if maybeName.MatchString(s) {
		nameMinLength := 1
		nameMaxLength := 256

		if len(s) < nameMinLength {
			return "", "", fmt.Errorf("invalid name: does not meet minimum length requirement of %d characters", nameMinLength)
		}

		if len(s) > nameMaxLength {
			return "", "", fmt.Errorf("invalid name: exceed maximum length requirement of %d characters", nameMaxLength)
		}

		return s, "", nil
	}

	address, err := mail.ParseAddress(s)
	if err != nil {
		return "", "", fmt.Errorf("invalid email: %q", s)
	}

	return "", address.Address, nil
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
		fmt.Fprintf(os.Stderr, "  Updated one time password for user %s\n", user.Email)
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

func promptUpdatePassword(oldPassword string) (string, error) {
	fmt.Fprintf(os.Stderr, "Enter a new password (min 8 characters)")

PROMPT:
	newPassword := ""
	if err := survey.AskOne(&survey.Password{Message: "New password:"}, &newPassword, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return "", err
	}

	if err := checkPasswordRequirements(newPassword, oldPassword); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		goto PROMPT
	}

	confirmNewPassword := ""
	if err := survey.AskOne(&survey.Password{Message: "Re-enter new password:"}, &confirmNewPassword, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return "", err
	}

	if confirmNewPassword != newPassword {
		fmt.Println("  Passwords do not match")
		goto PROMPT
	}

	return newPassword, nil
}

func checkPasswordRequirements(newPassword string, oldPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("  Password cannot be less than 8 characters")
	}
	if newPassword == oldPassword {
		return errors.New("  New password cannot equal your old password.")
	}
	return nil
}
