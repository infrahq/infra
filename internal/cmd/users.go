package cmd

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
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
	cmd.AddCommand(newUsersEditCmd(cli))
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
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			createResp, err := createUser(client, args[0], true)
			if err != nil {
				return err
			}

			cli.Output("Added user %q", args[0])

			if createResp.OneTimePassword != "" {
				cli.Output("Password: %s", createResp.OneTimePassword)
			}

			return nil
		},
	}

	return cmd
}

func newUsersEditCmd(cli *CLI) *cobra.Command {
	var editPassword bool

	cmd := &cobra.Command{
		Use:   "edit USER",
		Short: "Update a user",
		Example: `# Set a new one time password for a user
$ infra users edit janedoe@example.com --password`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !editPassword {
				return errors.New("Please specify a field to update. For options, run 'infra users edit --help'")
			}

			return updateUser(cli, args[0])
		},
	}

	cmd.Flags().BoolVar(&editPassword, "password", false, "Set a new one time password")

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

			logging.S.Debug("call server: list users")
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
	var force bool

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

			logging.S.Debugf("call server: list users named %q", name)
			users, err := client.ListUsers(api.ListUsersRequest{Name: name})
			if err != nil {
				return err
			}

			if users.Count == 0 && !force {
				return Error{Message: fmt.Sprintf("No user named %q ", name)}
			}

			if name == models.InternalInfraConnectorIdentityName {
				return Error{
					Message: "The \"connector\" user cannot be deleted, as it is not modifiable.",
				}
			}

			logging.S.Debugf("deleting %d users named %q...", users.Count, name)
			for _, user := range users.Items {
				logging.S.Debugf("...call server: delete user %s", user.ID)
				if err := client.DeleteUser(user.ID); err != nil {
					return err
				}

				cli.Output("Removed user %q", user.Name)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Exit successfully even if user does not exist")

	return cmd
}

// CreateUser creates an user within Infra
func CreateUser(req *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	client, err := defaultAPIClient()
	if err != nil {
		return nil, err
	}

	logging.S.Debugf("call server: create users named %q", req.Name)
	resp, err := client.CreateUser(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func updateUser(cli *CLI, name string) error {
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
		user, err := getUserByName(client, name)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return Error{Message: fmt.Sprintf("No user named %q in local provider; only local users can be edited", name)}
			}
			return err
		}

		req.ID = user.ID
	}

	fmt.Fprintf(cli.Stderr, "  Enter a new password (min. length 8):\n")
	req.Password, err = promptSetPassword(cli, "")
	if err != nil {
		return err
	}

	logging.S.Debugf("call server: update user %s", req.ID)
	if _, err := client.UpdateUser(req); err != nil {
		return err
	}

	if isSelf {
		cli.Output("  Updated password")
	} else {
		cli.Output("  Updated password for %q", name)
	}

	return nil
}

func getUserByName(client *api.Client, name string) (*api.User, error) {
	logging.S.Debugf("call server: list users named %q", name)
	users, err := client.ListUsers(api.ListUsersRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if users.Count == 0 {
		return nil, fmt.Errorf("unknown user %q", name)
	}

	if users.Count > 1 {
		return nil, fmt.Errorf("multiple results found for %q. check your server configurations", name)
	}

	return &users.Items[0], nil
}

func promptSetPassword(cli *CLI, oldPassword string) (string, error) {
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

	if err := survey.Ask(prompts, &passwordConfirm, cli.surveyIO); err != nil {
		return "", err
	}

	if passwordConfirm.Password != passwordConfirm.Confirm {
		cli.Output("  Passwords do not match. Please try again.")
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

// createUser creates a user with the requested name
func createUser(client *api.Client, name string, setOTP bool) (*api.CreateUserResponse, error) {
	logging.S.Debugf("call server: create user named %q", name)
	user, err := client.CreateUser(&api.CreateUserRequest{Name: name, SetOneTimePassword: setOTP})
	if err != nil {
		return nil, err
	}

	return user, nil
}
