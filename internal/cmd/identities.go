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

func newIdentitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identities",
		Aliases: []string{"id", "identity"},
		Short:   "Manage identities (users & machines)",
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newIdentitiesAddCmd())
	cmd.AddCommand(newIdentitiesEditCmd())
	cmd.AddCommand(newIdentitiesListCmd())
	cmd.AddCommand(newIdentitiesRemoveCmd())

	return cmd
}

type identityOptions struct {
	Password bool `mapstructure:"password"`
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
			name := args[0]

			var options identityOptions
			if err := parseOptions(cmd, &options, ""); err != nil {
				return err
			}

			createResp, err := CreateLocalIdentity(name)
			if err != nil {
				return err
			}

			if createResp.OneTimePassword != "" {
				fmt.Fprintf(os.Stderr, "Created user identity.\n")
				fmt.Printf("Email: %s\n", createResp.Name)
				fmt.Printf("Password: %s\n", createResp.OneTimePassword)
			} else {
				fmt.Fprintf(os.Stderr, "Created machine identity.\n")
				fmt.Printf("Name: %s\n", createResp.Name)
			}

			return nil
		},
	}

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
			if err := parseOptions(cmd, &options, ""); err != nil {
				return err
			}

			kind, err := checkUserOrMachine(name)
			if err != nil {
				return err
			}

			if kind == models.MachineKind {
				fmt.Println("machine identities have no editable fields")
			}

			if kind == models.UserKind {
				if !options.Password {
					return errors.New("specify the --password flag to update the password")
				}

				newPassword, err := promptUpdatePassword("")
				if err != nil {
					return err
				}

				err = UpdateIdentity(name, newPassword)
				if err != nil {
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
				Name     string `header:"Name"`
				Type     string `header:"Type"`
				Provider string `header:"Provider"`
			}

			var rows []row

			identities, err := client.ListIdentities(api.ListIdentitiesRequest{})
			if err != nil {
				return err
			}

			providers := make(map[uid.ID]string)

			for _, identity := range identities {
				if providers[identity.ProviderID] == "" {
					p, err := client.GetProvider(identity.ProviderID)
					if err != nil {
						logging.S.Debugf("unable to retrieve identity provider: %s", err)
					} else {
						providers[p.ID] = p.Name
					}
				}

				rows = append(rows, row{
					Name:     identity.Name,
					Type:     identity.Kind,
					Provider: providers[identity.ProviderID],
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
			if err := parseOptions(cmd, &options, ""); err != nil {
				return err
			}

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

	_, err := mail.ParseAddress(s)
	if err != nil {
		return models.UserKind, fmt.Errorf("invalid email: %q", s)
	}

	return models.UserKind, nil
}

// Creates a user for the local identity provider
func CreateLocalIdentity(name string) (*api.CreateIdentityResponse, error) {
	client, err := defaultAPIClient()
	if err != nil {
		return nil, err
	}

	kind, err := checkUserOrMachine(name)
	if err != nil {
		return nil, err
	}

	infraProvider, err := GetProviderByName(client, models.InternalInfraProviderName)
	if err != nil {
		logging.S.Debug(err)
		return nil, fmt.Errorf("no infra provider found, to manage local identities create a local provider named 'infra'")
	}

	resp, err := client.CreateIdentity(&api.CreateIdentityRequest{Name: name, ProviderID: infraProvider.ID, Kind: kind.String()})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func UpdateIdentity(name, newPassword string) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	infraProvider, err := GetProviderByName(client, models.InternalInfraProviderName)
	if err != nil {
		logging.S.Debug(err)
		return fmt.Errorf("no infra provider found, to manage local users create a local provider named 'infra'")
	}

	user := &api.Identity{}

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
		user, err = GetIdentityFromName(client, name, infraProvider)
		if err != nil {
			if errors.Is(err, ErrIdentityNotFound) {
				return fmt.Errorf("Identity %s not found in local provider; only local identities can be edited", name)
			}
			return err
		}
		// Todo otp: update term to temporary password (https://github.com/infrahq/infra/issues/1441)
		fmt.Fprintf(os.Stderr, "  Updated one time password for user %s.\n", user.Name)
	}

	if _, err := client.UpdateIdentity(&api.UpdateIdentityRequest{ID: user.ID, Password: newPassword}); err != nil {
		return err
	}

	return nil
}

func GetIdentityFromName(client *api.Client, name string, provider *api.Provider) (*api.Identity, error) {
	users, err := client.ListIdentities(api.ListIdentitiesRequest{Name: name, ProviderID: provider.ID})
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
	fmt.Fprintf(os.Stderr, "  Enter a new password (min length 8):\n")

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
		fmt.Println("  Passwords do not match.")
		goto PROMPT
	}

	return newPassword, nil
}

func checkPasswordRequirements(newPassword string, oldPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("  Password cannot be less than 8 characters.")
	}
	if newPassword == oldPassword {
		return errors.New("  New password cannot be the same as your old password.")
	}
	return nil
}
