package cmd

import (
	"errors"
	"fmt"
	"os"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

func newUsersCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Short:   "Manage user identities",
		Aliases: []string{"user"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newUsersAddCmd(cli))
	cmd.AddCommand(newUsersEditCmd())
	cmd.AddCommand(newUsersListCmd(cli))
	cmd.AddCommand(newUsersRemoveCmd(cli))

	return cmd
}

func newUsersAddCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add USER",
		Short: "Create a user.",
		Long: `Create a user.

Note: A new user must change their one time password before further usage.`,
		Args: ExactArgs(1),
		Example: `# Create a user
$ infra users add johndoe@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			createResp, err := CreateUser(&api.CreateUserRequest{Name: name, SetOneTimePassword: true})
			if err != nil {
				return err
			}

			fmt.Fprintf(cli.Stderr, "User created.\n")
			cli.Output("Name: %s", createResp.Name)

			if createResp.OneTimePassword != "" {
				cli.Output("Password: %s", createResp.OneTimePassword)
			}

			return nil
		},
	}

	return cmd
}

type editUserCmdOptions struct {
	Password       bool
	NonInteractive bool
}

func newUsersEditCmd() *cobra.Command {
	var opts editUserCmdOptions
	cmd := &cobra.Command{
		Use:   "edit USER",
		Short: "Update a user",
		Example: `# Set a new one time password for a user
$ infra users edit janedoe@example.com --password`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if !opts.Password {
				return errors.New("Please specify a field to update. For options, run 'infra users edit --help'")
			}

			if opts.Password && opts.NonInteractive {
				return errors.New("Non-interactive mode is not supported to edit sensitive fields.")
			}

			return UpdateUser(name, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Password, "password", "p", false, "Set a new one time password")
	addNonInteractiveFlag(cmd.Flags(), &opts.NonInteractive)

	return cmd
}

func newUsersListCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List users",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			type row struct {
				Name       string `header:"Name"`
				LastSeenAt string `header:"Last Seen"`
			}

			var rows []row

			users, err := client.ListUsers(api.ListUsersRequest{})
			if err != nil {
				return err
			}

			for _, user := range users.Items {
				rows = append(rows, row{
					Name:       user.Name,
					LastSeenAt: user.LastSeenAt.Relative("never"),
				})
			}

			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No users found")
			}

			return nil
		},
	}
}

func newUsersRemoveCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove USER",
		Aliases: []string{"rm"},
		Short:   "Delete a user",
		Example: `# Delete a user
$ infra users remove janedoe@example.com`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			users, err := client.ListUsers(api.ListUsersRequest{Name: name})
			if err != nil {
				return err
			}

			for _, user := range users.Items {
				err := client.DeleteUser(user.ID)
				if err != nil {
					return err
				}

				cli.Output("Removed user %q", name)
			}

			return nil
		},
	}

	return cmd
}

// CreateUser creates an user within infra
func CreateUser(req *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	client, err := defaultAPIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.CreateUser(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func UpdateUser(name string, cmdOptions editUserCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	req := &api.UpdateUserRequest{}

	isSelf, err := isUserSelf(name)
	if err != nil {
		return err
	}

	if isSelf {
		config, err := currentHostConfig()
		if err != nil {
			return err
		}

		if req.ID, err = config.PolymorphicID.ID(); err != nil {
			return err
		}
	} else {
		user, err := GetUserByName(client, name)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return fmt.Errorf("User %s not found in local provider; only local users can be edited", name)
			}
			return err
		}

		req.ID = user.ID
	}

	if cmdOptions.Password {
		fmt.Fprintf(os.Stderr, "  Enter a new one time password (min length 8):\n")
		req.Password, err = promptSetPassword("")
		if err != nil {
			return err
		}
	}

	if _, err := client.UpdateUser(req); err != nil {
		return err
	}

	if !isSelf {
		// Todo otp: update term to temporary password (https://github.com/infrahq/infra/issues/1441)
		fmt.Fprintf(os.Stderr, "  Updated one time password for user.\n")
	}

	return nil
}

func GetUserByName(client *api.Client, name string) (*api.User, error) {
	users, err := client.ListUsers(api.ListUsersRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if users.Count == 0 {
		return nil, ErrUserNotFound
	}

	if users.Count != 1 {
		return nil, fmt.Errorf("invalid users response, there should only be one user that matches a name, but multiple were found")
	}

	return &users.Items[0], nil
}

func promptSetPassword(oldPassword string) (string, error) {
	var passwordConfirm struct {
		Password string
		Confirm  string
	}

PROMPT:
	prompts := []*survey.Question{
		{
			Name:     "Password",
			Prompt:   &survey.Password{Message: "Password:"},
			Validate: checkPasswordRequirements(oldPassword),
		},
		{
			Name:     "Confirm",
			Prompt:   &survey.Password{Message: "Confirm Password:"},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(prompts, &passwordConfirm, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return "", err
	}

	if passwordConfirm.Password != passwordConfirm.Confirm {
		fmt.Println("  Passwords do not match. Please try again.")
		goto PROMPT
	}

	return passwordConfirm.Password, nil
}

func checkPasswordRequirements(oldPassword string) survey.Validator {
	return func(val interface{}) error {
		newPassword, ok := val.(string)
		if !ok {
			return fmt.Errorf("unexpected type for password: %T", val)
		}

		if len(newPassword) < 8 {
			return fmt.Errorf("input must be at least 8 characters long")
		}

		if newPassword == oldPassword {
			return fmt.Errorf("input must be different than the current password")
		}

		return nil
	}
}

// isUserSelf checks if the caller is updating their current local user
func isUserSelf(name string) (bool, error) {
	config, err := currentHostConfig()
	if err != nil {
		return false, err
	}

	return config.Name == name, nil
}
