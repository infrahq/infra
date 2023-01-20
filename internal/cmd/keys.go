package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/format"
	"github.com/infrahq/infra/internal/logging"
)

const (
	thirtyDays = 30 * (24 * time.Hour)
	oneYear    = 365 * (24 * time.Hour)
)

func newKeysCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Short:   "Manage access keys",
		Aliases: []string{"key"},
		GroupID: groupManagement,
	}

	cmd.AddCommand(newKeysListCmd(cli))
	cmd.AddCommand(newKeysAddCmd(cli))
	cmd.AddCommand(newKeysRemoveCmd(cli))

	return cmd
}

type keyCreateOptions struct {
	Name              string
	UserName          string
	Expiry            time.Duration
	InactivityTimeout time.Duration
	Connector         bool
	Quiet             bool
}

func newKeysAddCmd(cli *CLI) *cobra.Command {
	var options keyCreateOptions

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create an access key",
		Long:  `Create an access key for a user or a connector.`,
		Args:  NoArgs,
		Example: `
# Create an access key named 'example-key' for a user that expires in 12 hours
$ infra keys add --expiry=12h --name example-key

# Create an access key to add a Kubernetes connection to Infra
$ infra keys add --connector

# Set an environment variable with the newly created access key
$ MY_ACCESS_KEY=$(infra keys add -q --name my-key)
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if options.Name != "" {
				if strings.Contains(options.Name, " ") {
					return Error{Message: "Key name cannot contain spaces"}
				}
			}

			client, err := cli.apiClient()
			if err != nil {
				return err
			}

			config, err := currentHostConfig()
			if err != nil {
				return err
			}

			userID := config.UserID

			// override the user setting if the user wants to create a connector access key
			if options.Connector {
				options.UserName = "connector"
			}

			if options.UserName != "" {
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

			logging.Debugf("call server: create access key named %q", options.Name)
			resp, err := client.CreateAccessKey(ctx, &api.CreateAccessKeyRequest{
				UserID:            userID,
				Name:              options.Name,
				Expiry:            api.Duration(options.Expiry),
				InactivityTimeout: api.Duration(options.InactivityTimeout),
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

			if options.Quiet {
				cli.Output(resp.AccessKey)
				return nil
			}

			var expMsg strings.Builder
			expMsg.WriteString("This key will expire in ")
			if options.Expiry == oneYear {
				expMsg.WriteString("1 year")
			} else {
				expMsg.WriteString(format.ExactDuration(options.Expiry))
			}
			if !resp.Expires.Equal(resp.InactivityTimeout) {
				expMsg.WriteString(", and must be used every ")
				if options.InactivityTimeout == thirtyDays {
					expMsg.WriteString("30 days")
				} else {
					expMsg.WriteString(format.ExactDuration(options.InactivityTimeout))
				}
				expMsg.WriteString(" to remain valid")
			}
			if options.UserName != "" {
				cli.Output("Issued access key %q for %q", resp.Name, options.UserName)
			} else {
				cli.Output("Issued access key %q", resp.Name)
			}
			cli.Output(expMsg.String())
			cli.Output("")

			cli.Output("Key: %s", resp.AccessKey)
			return nil
		},
	}

	cmd.Flags().StringVar(&options.Name, "name", "", "The name of the access key")
	cmd.Flags().StringVar(&options.UserName, "user", "", "The name of the user who will own the key")
	cmd.Flags().BoolVar(&options.Connector, "connector", false, "Create the key for the connector")
	cmd.Flags().BoolVarP(&options.Quiet, "quiet", "q", false, "Only display the access key")
	cmd.Flags().DurationVar(&options.Expiry, "expiry", oneYear, "The total time that the access key will be valid for")
	cmd.Flags().DurationVar(&options.InactivityTimeout, "inactivity-timeout", thirtyDays, "A specified deadline that the access key must be used within to remain valid")

	return cmd
}

type keyRemoveOptions struct {
	Force     bool
	UserName  string
	Connector bool
}

func newKeysRemoveCmd(cli *CLI) *cobra.Command {
	var options keyRemoveOptions

	cmd := &cobra.Command{
		Use:     "remove KEY",
		Aliases: []string{"rm"},
		Short:   "Delete an access key",
		Args:    ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cli.apiClient()
			if err != nil {
				return err
			}

			config, err := currentHostConfig()
			if err != nil {
				return err
			}

			keyName := args[0]
			userID := config.UserID

			// override the user setting if the user wants to delete a connector access key
			if options.Connector {
				options.UserName = "connector"
			}

			if options.UserName != "" {
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

			keys, err := listAll(ctx, client.ListAccessKeys, api.ListAccessKeysRequest{Name: keyName, UserID: userID})
			if err != nil {
				return err
			}

			if len(keys) == 0 {
				// the username for the user this is being run against is used in the error
				username := options.UserName
				if username == "" {
					username = config.Name
				}
				errMsg := fmt.Sprintf("Access key %q for user %q does not exist", keyName, username)
				if username == config.Name {
					// give a suggestion on how to use a different key if running the command on yourself
					errMsg = fmt.Sprintf("%s\nUse the '--user' flag to remove an access key for a different user", errMsg)
				}
				return Error{
					Message: errMsg,
				}
			}

			err = client.DeleteAccessKey(ctx, keys[0].ID)
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

	cmd.Flags().BoolVar(&options.Force, "force", false, "Exit successfully even if access key does not exist")
	cmd.Flags().StringVar(&options.UserName, "user", "", "The name of the user who owns the key")
	cmd.Flags().BoolVar(&options.Connector, "connector", false, "Remove a key for the connector")

	return cmd
}

type keyListOptions struct {
	AllUsers    bool
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
			client, err := cli.apiClient()
			if err != nil {
				return err
			}

			ctx := context.Background()

			config, err := currentHostConfig()
			if err != nil {
				return err
			}

			var keys []api.AccessKey
			userID := config.UserID

			if options.UserName != "" {
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

			if options.AllUsers {
				userID = 0
			}

			logging.Debugf("call server: list access keys")
			keys, err = listAll(ctx, client.ListAccessKeys, api.ListAccessKeysRequest{ShowExpired: options.ShowExpired, UserID: userID})
			if err != nil {
				return err
			}

			// TODO: remove IssuedFor unless the user is getting access keys for a different user or using --all
			type row struct {
				Name              string `header:"NAME"`
				IssuedFor         string `header:"ISSUED FOR"`
				Created           string `header:"CREATED"`
				LastUsed          string `header:"LAST USED"`
				Expires           string `header:"EXPIRES"`
				InactivityTimeout string `header:"INACTIVITY TIMEOUT"`
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
					LastUsed:          format.HumanTime(k.LastUsed.Time(), "never"),
					Expires:           format.HumanTime(k.Expires.Time(), "never"),
					InactivityTimeout: format.HumanTime(k.InactivityTimeout.Time(), "never"),
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

	cmd.Flags().BoolVar(&options.AllUsers, "all", false, "Show keys for all users")
	cmd.Flags().StringVar(&options.UserName, "user", "", "The name of a user to list access keys for")
	cmd.Flags().BoolVar(&options.ShowExpired, "show-expired", false, "Show expired access keys")
	return cmd
}
