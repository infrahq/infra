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
	"github.com/infrahq/infra/internal/server/models"
)

func newIdentitiesCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identities",
		Aliases: []string{"id", "identity"},
		Short:   "Manage identities (users & machines)",
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newIdentitiesAddCmd(cli))
	cmd.AddCommand(newIdentitiesEditCmd())
	cmd.AddCommand(newIdentitiesListCmd(cli))
	cmd.AddCommand(newIdentitiesRemoveCmd(cli))

	return cmd
}

func newIdentitiesAddCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add IDENTITY",
		Short: "Create an identity.",
		Long: `Create an identity.

If a valid email is detected, a user identity is created. 
If a username is detected, a machine identity is created.

A new user identity must change their one time password before further usage.`,
		Args: ExactArgs(1),
		Example: `# Create a local user
$ infra identities add johndoe@example.com

# Create a machine
$ infra identities add machine-a`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			createResp, err := CreateIdentity(&api.CreateIdentityRequest{Name: name, SetOneTimePassword: true})
			if err != nil {
				return err
			}

			if createResp.OneTimePassword != "" {
				fmt.Fprintf(cli.Stderr, "Created user identity.\n")
				cli.Output("Email: %s", createResp.Name)
				cli.Output("Password: %s", createResp.OneTimePassword)
			} else {
				fmt.Fprintf(cli.Stderr, "Created machine identity.\n")
				cli.Output("Name: %s", createResp.Name)
			}

			return nil
		},
	}

	return cmd
}

type editIdentityCmdOptions struct {
	Password       bool
	NonInteractive bool
}

func newIdentitiesEditCmd() *cobra.Command {
	var opts editIdentityCmdOptions
	cmd := &cobra.Command{
		Use:   "edit IDENTITY",
		Short: "Update an identity",
		Example: `# Set a new one time password for a local user
$ infra identities edit janedoe@example.com --password`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			kind, err := checkUserOrMachine(name)
			if err != nil {
				return err
			}

			if kind == models.MachineKind {
				return fmt.Errorf("Machine identities cannot be edited.")
			}

			if kind == models.UserKind {
				if !opts.Password {
					return errors.New("Please specify a field to update. For options, run 'infra identities edit --help'")
				}

				if opts.Password && opts.NonInteractive {
					return errors.New("Non-interactive mode is not supported to edit sensitive fields.")
				}

				if err = UpdateIdentity(name, opts); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.Password, "password", "p", false, "Set a new one time password")
	addNonInteractiveFlag(cmd.Flags(), &opts.NonInteractive)

	return cmd
}

func newIdentitiesListCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List identities",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			type row struct {
				Name       string `header:"Name"`
				Type       string `header:"Type"`
				LastSeenAt string `header:"Last Seen"`
			}

			var rows []row

			identities, err := client.ListIdentities(api.ListIdentitiesRequest{})
			if err != nil {
				return err
			}

			for _, identity := range identities {
				rows = append(rows, row{
					Name:       identity.Name,
					Type:       identity.Kind,
					LastSeenAt: identity.LastSeenAt.Relative("never"),
				})
			}

			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No identities found")
			}

			return nil
		},
	}
}

func newIdentitiesRemoveCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove IDENTITY",
		Aliases: []string{"rm"},
		Short:   "Delete an identity",
		Example: `# Delete a local user
$ infra identities remove janedoe@example.com

# Delete a machine
$ infra identities remove machine-a`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			identities, err := client.ListIdentities(api.ListIdentitiesRequest{Name: name})
			if err != nil {
				return err
			}

			for _, identity := range identities {
				err := client.DeleteIdentity(identity.ID)
				if err != nil {
					return err
				}
			}

			fmt.Fprintf(cli.Stderr, "Deleted identity %q\n", name)

			return nil
		},
	}

	return cmd
}

// checkUserOrMachine infers whether the input s specifies a user identity (email) or a machine
// identity (name). The input is considered a name if it has the format `^[a-zA-Z0-9-_/]+$`.
// All other input formats will be considered an email. If an email input fails validation,
// return an error.
func checkUserOrMachine(s string) (models.IdentityKind, error) {
	maybeName := regexp.MustCompile("^[a-zA-Z0-9-_./]+$")
	if maybeName.MatchString(s) {
		nameMinLength := 1
		nameMaxLength := 256

		if len(s) < nameMinLength {
			return models.MachineKind, fmt.Errorf("invalid name: does not meet minimum length requirement of %d characters", nameMinLength)
		}

		if len(s) > nameMaxLength {
			return models.MachineKind, fmt.Errorf("invalid name: exceed maximum length requirement of %d characters", nameMaxLength)
		}

		return models.MachineKind, nil
	}

	if err := checkEmailRequirements(s); err != nil {
		return models.MachineKind, err
	}

	return models.UserKind, nil
}

// CreateIdentity creates an identity within infra
func CreateIdentity(req *api.CreateIdentityRequest) (*api.CreateIdentityResponse, error) {
	client, err := defaultAPIClient()
	if err != nil {
		return nil, err
	}

	kind, err := checkUserOrMachine(req.Name)
	if err != nil {
		return nil, err
	}

	req.Kind = kind.String()

	resp, err := client.CreateIdentity(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func UpdateIdentity(name string, cmdOptions editIdentityCmdOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	req := &api.UpdateIdentityRequest{}

	isSelf, err := isIdentitySelf(name)
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
		user, err := GetIdentityFromName(client, name)
		if err != nil {
			if errors.Is(err, ErrIdentityNotFound) {
				return fmt.Errorf("Identity %s not found in local provider; only local identities can be edited", name)
			}
			return err
		}

		req.ID = user.ID
	}

	if cmdOptions.Password {
		req.Password, err = promptUpdatePassword("")
		if err != nil {
			return err
		}
	}

	if _, err := client.UpdateIdentity(req); err != nil {
		return err
	}

	if !isSelf {
		// Todo otp: update term to temporary password (https://github.com/infrahq/infra/issues/1441)
		fmt.Fprintf(os.Stderr, "  Updated one time password for user.\n")
	}

	return nil
}

func GetIdentityFromName(client *api.Client, name string) (*api.Identity, error) {
	users, err := client.ListIdentities(api.ListIdentitiesRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrIdentityNotFound
	}

	if len(users) != 1 {
		return nil, fmt.Errorf("invalid users response, there should only be one user that matches a name, but multiple were found")
	}

	return &users[0], nil
}

func promptUpdatePassword(oldPassword string) (string, error) {
	fmt.Fprintf(os.Stderr, "  Enter a new one time password (min length 8):\n")

	newPassword, err := promptPasswordConfirm(oldPassword)
	if err != nil {
		return "", err
	}

	return newPassword, nil
}

func promptPasswordConfirm(oldPassword string) (string, error) {
	var passwordConfirm struct {
		Password string
		Confirm  string
	}

	prompts := []*survey.Question{
		{
			Name:     "Password",
			Prompt:   &survey.Password{Message: "Password:"},
			Validate: checkPasswordRequirements(oldPassword),
		},
		{
			Name:     "Confirm",
			Prompt:   &survey.Password{Message: "Confirm Password:"},
			Validate: checkConfirmPassword(&passwordConfirm.Password),
		},
	}

	if err := survey.Ask(prompts, &passwordConfirm, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return "", err
	}

	return passwordConfirm.Password, nil
}

func checkEmailRequirements(val interface{}) error {
	email, ok := val.(string)
	if !ok {
		return fmt.Errorf("unexpected type for email: %T", val)
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("input must be a valid email")
	}

	return nil
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

func checkConfirmPassword(password *string) survey.Validator {
	return func(val interface{}) error {
		confirm, ok := val.(string)
		if !ok {
			return fmt.Errorf("unexpected type for password: %T", val)
		}

		if *password != confirm {
			return fmt.Errorf("input must match the new password")
		}

		return nil
	}
}

// isIdentitySelf checks if the caller is updating their current local user
func isIdentitySelf(name string) (bool, error) {
	config, err := currentHostConfig()
	if err != nil {
		return false, err
	}

	return config.Name == name, nil
}
