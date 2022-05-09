package cmd

import (
	"errors"
	"fmt"
	"os"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

func newIdentitiesCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identities",
		Aliases: []string{"id", "identity"},
		Short:   "Manage user identities",
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
		Short: "Create an identity",
		Long: `Create an identity

Note: A new user identity must change their one time password before further usage.`,
		Args: ExactArgs(1),
		Example: `# Create a user
$ infra identities add johndoe@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			createResp, err := CreateIdentity(&api.CreateIdentityRequest{Name: name, SetOneTimePassword: true})
			if err != nil {
				return err
			}

			fmt.Fprintf(cli.Stderr, "Identity created.\n")
			cli.Output("Name: %s", createResp.Name)

			if createResp.OneTimePassword != "" {
				cli.Output("Password: %s", createResp.OneTimePassword)
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
		Example: `# Set a new one time password for an identity
$ infra identities edit janedoe@example.com --password`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if !opts.Password {
				return errors.New("Please specify a field to update. For options, run 'infra identities edit --help'")
			}

			if opts.Password && opts.NonInteractive {
				return errors.New("Non-interactive mode is not supported to edit sensitive fields.")
			}

			return UpdateIdentity(name, opts)
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
		Example: `# Delete an identity
$ infra identities remove janedoe@example.com`,
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

// CreateIdentity creates an identity within infra
func CreateIdentity(req *api.CreateIdentityRequest) (*api.CreateIdentityResponse, error) {
	client, err := defaultAPIClient()
	if err != nil {
		return nil, err
	}

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
		user, err := GetIdentityByName(client, name)
		if err != nil {
			if errors.Is(err, ErrIdentityNotFound) {
				return fmt.Errorf("Identity %s not found in local provider; only local identities can be edited", name)
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

	if _, err := client.UpdateIdentity(req); err != nil {
		return err
	}

	if !isSelf {
		// Todo otp: update term to temporary password (https://github.com/infrahq/infra/issues/1441)
		fmt.Fprintf(os.Stderr, "  Updated one time password for identity.\n")
	}

	return nil
}

func GetIdentityByName(client *api.Client, name string) (*api.Identity, error) {
	users, err := client.ListIdentities(api.ListIdentitiesRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrIdentityNotFound
	}

	if len(users) != 1 {
		return nil, fmt.Errorf("invalid identities response, there should only be one identity that matches a name, but multiple were found")
	}

	return &users[0], nil
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
			Validate: checkConfirmPassword(),
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

func checkConfirmPassword() survey.Validator {
	return func(val interface{}) error {
		_, ok := val.(string)
		if !ok {
			return fmt.Errorf("unexpected type for password: %T", val)
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
