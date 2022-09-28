package cmd

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/format"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
)

const thirtyDays = 30 * (24 * time.Hour)

func newKeysCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Short:   "Manage access keys",
		Aliases: []string{"key"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newKeysListCmd(cli))
	cmd.AddCommand(newKeysAddCmd(cli))
	cmd.AddCommand(newKeysRemoveCmd(cli))

	return cmd
}

type keyCreateOptions struct {
	Name              string
	TTL               time.Duration
	ExtensionDeadline time.Duration
}

func newKeysAddCmd(cli *CLI) *cobra.Command {
	var options keyCreateOptions

	cmd := &cobra.Command{
		Use:   "add USER|connector",
		Short: "Create an access key",
		Long:  `Create an access key for a user or a connector.`,
		Example: `
# Create an access key named 'example-key' for a user that expires in 12 hours
$ infra keys add user@example.com --ttl=12h --name example-key

# Create an access key to add a Kubernetes connection to Infra
$ infra keys add connector
`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userName := args[0]

			if options.Name != "" {
				if strings.Contains(options.Name, " ") {
					return Error{Message: "Key name cannot contain spaces"}
				}
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			config, err := currentHostConfig()
			if err != nil {
				return err
			}

			var userID uid.ID
			if userName == config.Name { // user is requesting their own stuff
				userID = config.UserID
			} else {
				user, err := getUserByNameOrID(client, userName)
				if err != nil {
					if api.ErrorStatusCode(err) == 403 {
						logging.Debugf("%s", err.Error())
						return Error{
							Message: "Cannot list keys: missing privileges for GetUser",
						}
					}
					return err
				}
				userID = user.ID
			}

			logging.Debugf("call server: create access key named %q", options.Name)
			resp, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{
				UserID:            userID,
				Name:              options.Name,
				TTL:               api.Duration(options.TTL),
				ExtensionDeadline: api.Duration(options.ExtensionDeadline),
			})
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.Debugf("%s", err.Error())
					return Error{
						Message: "Cannot create key: missing privileges for CreateKey",
					}
				}
				return err
			}

			var expMsg strings.Builder
			expMsg.WriteString("This key will expire in ")
			expMsg.WriteString(format.ExactDuration(options.TTL))
			if !resp.Expires.Equal(resp.ExtensionDeadline) {
				expMsg.WriteString(", and must be used every ")
				expMsg.WriteString(format.ExactDuration(options.ExtensionDeadline))
				expMsg.WriteString(" to remain valid")
			}
			cli.Output("Issued access key %q for %q", resp.Name, userName)
			cli.Output(expMsg.String())
			cli.Output("")

			cli.Output("Key: %s", resp.AccessKey)
			return nil
		},
	}

	cmd.Flags().StringVar(&options.Name, "name", "", "The name of the access key")
	cmd.Flags().DurationVar(&options.TTL, "ttl", thirtyDays, "The total time that the access key will be valid for")
	cmd.Flags().DurationVar(&options.ExtensionDeadline, "extension-deadline", thirtyDays, "A specified deadline that the access key must be used within to remain valid")

	return cmd
}

func newKeysRemoveCmd(cli *CLI) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "remove KEY",
		Aliases: []string{"rm"},
		Short:   "Delete an access key",
		Args:    ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			err = client.DeleteAccessKeyByName(args[0])
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.Debugf("%s", err.Error())
					return Error{
						Message: "Cannot delete key: missing privileges for DeleteKey",
					}
				}
				return err
			}

			cli.Output("Removed access key %q", args[0])

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Exit successfully even if access key does not exist")

	return cmd
}

type keyListOptions struct {
	UserName    string
	ShowExpired bool
}

func newKeysListCmd(cli *CLI) *cobra.Command {
	var options keyListOptions

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List access keys",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			var keys []api.AccessKey
			var userID uid.ID
			if options.UserName != "" {
				config, err := currentHostConfig()
				if err != nil {
					return err
				}

				if options.UserName == config.Name { // user is requesting their own stuff
					userID = config.UserID
				} else {
					user, err := getUserByNameOrID(client, options.UserName)
					if err != nil {
						if api.ErrorStatusCode(err) == 403 {
							logging.Debugf("%s", err.Error())
							return Error{
								Message: "Cannot list keys: missing privileges for GetUser",
							}
						}
						return err
					}
					userID = user.ID
				}
			}

			logging.Debugf("call server: list access keys")
			keys, err = listAll(client.ListAccessKeys, api.ListAccessKeysRequest{ShowExpired: options.ShowExpired, UserID: userID})
			if err != nil {
				return err
			}

			type row struct {
				Name              string `header:"NAME"`
				IssuedFor         string `header:"ISSUED FOR"`
				Created           string `header:"CREATED"`
				Expires           string `header:"EXPIRES"`
				ExtensionDeadline string `header:"EXTENSION DEADLINE"`
			}

			var rows []row
			for _, k := range keys {
				name := k.IssuedFor.String()
				if k.IssuedForName != "" {
					name = k.IssuedForName
				}
				rows = append(rows, row{
					Name:              k.Name,
					IssuedFor:         name,
					Created:           format.HumanTime(k.Created.Time(), "never"),
					Expires:           format.HumanTime(k.Expires.Time(), "never"),
					ExtensionDeadline: format.HumanTime(k.ExtensionDeadline.Time(), "never"),
				})
			}

			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No access keys found")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&options.UserName, "user", "", "The name of a user to list access keys for")
	cmd.Flags().BoolVar(&options.ShowExpired, "show-expired", false, "Show expired access keys")
	return cmd
}
