package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
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
		Short: "Create a user",
		Long: `Create a user.

Note: A temporary password will be created. The user will be prompted to set a new password on first login.`,
		Args: ExactArgs(1),
		Example: `# Create a user
$ infra users add johndoe@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			_, err := mail.ParseAddress(email)
			if err != nil {
				return fmt.Errorf("username must be a valid email")
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			createResp, err := createUser(client, args[0])
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.Debugf("%s", err.Error())
					return Error{
						Message: "Cannot add users: missing privileges for CreateUser",
					}
				}
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
		Example: `# Set a new password for a user
$ infra users edit janedoe@example.com --password`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !editPassword {
				return errors.New("Please specify a field to update. For options, run 'infra users edit --help'")
			}

			return updateUser(cli, args[0])
		},
	}

	cmd.Flags().BoolVar(&editPassword, "password", false, "Set a new password")

	return cmd
}

func newUsersListCmd(cli *CLI) *cobra.Command {
	var format string

	cmd := &cobra.Command{
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
				Providers  string `header:"Provided By"`
			}

			var rows []row

			logging.Debugf("call server: list users")
			users, err := client.ListUsers(api.ListUsersRequest{})
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.Debugf("%s", err.Error())
					return Error{
						Message: "Cannot list users: missing privileges for ListUsers",
					}
				}
				return err
			}

			switch format {
			case "json":
				jsonOutput, err := json.Marshal(users)
				if err != nil {
					return err
				}
				cli.Output(string(jsonOutput))
			default:
				for _, user := range users.Items {
					rows = append(rows, row{
						Name:       user.Name,
						LastSeenAt: HumanTime(user.LastSeenAt.Time(), "never"),
						Providers:  strings.Join(user.ProviderNames, ", "),
					})
				}

				if len(rows) > 0 {
					printTable(rows, cli.Stdout)
				} else {
					cli.Output("No users found")
				}
			}

			return nil
		},
	}

	addFormatFlag(cmd.Flags(), &format)
	return cmd
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

			logging.Debugf("call server: list users named %q", name)
			users, err := client.ListUsers(api.ListUsersRequest{Name: name})
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.Debugf("%s", err.Error())
					return Error{
						Message: "Cannot delete users: missing privileges for ListUsers",
					}
				}
				return err
			}

			if users.Count == 0 && !force {
				return Error{Message: fmt.Sprintf("No user named %q ", name)}
			}

			logging.Debugf("deleting %d users named %q...", users.Count, name)
			for _, user := range users.Items {
				logging.Debugf("...call server: delete user %s", user.ID)
				if err := client.DeleteUser(user.ID); err != nil {
					if api.ErrorStatusCode(err) == 403 {
						logging.Debugf("%s", err.Error())
						return Error{
							Message: "Cannot delete users: missing privileges for DeleteUsers",
						}
					}
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

	logging.Debugf("call server: create users named %q", req.Name)
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

	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	isSelf, err := isUserSelf(name)
	if err != nil {
		return err
	}

	if isSelf {
		req := &api.UpdateUserRequest{ID: config.UserID}

		fmt.Fprintf(cli.Stderr, "  Enter a new password (min. length 8):\n")
		req.Password, err = promptSetPassword(cli, "")
		if err != nil {
			return err
		}

		if _, err := client.UpdateUser(req); err != nil {
			return err
		}

		cli.Output("  Updated password")

		return nil
	}

	ok, err := hasAccessToChangePasswordsForOtherUsers(client, config)
	if err != nil {
		return err
	}
	if !ok {
		return Error{Message: "No permission to change password for user " + name}
	}

	user, err := getUserByName(client, name)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			logging.Debugf("user not found: %s", err)
			return Error{Message: fmt.Sprintf("No user named %q in local provider; only local users can be edited", name)}
		} else if api.ErrorStatusCode(err) == 403 {
			logging.Debugf("%s", err.Error())
			return Error{
				Message: fmt.Sprintf("Cannot update user %q: missing privileges for GetUser", name),
			}
		}

		return err
	}

	tmpPassword, err := generate.CryptoRandom(12, generate.CharsetPassword)
	if err != nil {
		return err
	}

	if _, err := client.UpdateUser(&api.UpdateUserRequest{
		ID:       user.ID,
		Password: tmpPassword,
	}); err != nil {
		return err
	}

	cli.Output("  Temporary password for user %q set to: %s", name, tmpPassword)

	return nil
}

func getUserByName(client *api.Client, name string) (*api.User, error) {
	users, err := client.ListUsers(api.ListUsersRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if users.Count == 0 {
		return nil, fmt.Errorf("%w: unknown user %q", ErrUserNotFound, name)
	}

	if users.Count > 1 {
		logging.Errorf("multiple users matching name %q. Likely missing database index on identities(name)", name)
		return nil, fmt.Errorf("multiple users matching name %q", name)
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
func createUser(client *api.Client, name string) (*api.CreateUserResponse, error) {
	logging.Debugf("call server: create user named %q", name)
	user, err := client.CreateUser(&api.CreateUserRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// check if the user has permissions to reset passwords for another user.
// This might be handy for customizing error messages
func hasAccessToChangePasswordsForOtherUsers(client *api.Client, config *ClientHostConfig) (bool, error) {
	grants, err := client.ListGrants(api.ListGrantsRequest{
		User:          config.UserID,
		Privilege:     api.InfraAdminRole,
		Resource:      "infra",
		ShowInherited: true,
	})
	if err != nil {
		return false, err
	}

	return len(grants.Items) > 0, nil
}
