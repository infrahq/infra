package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

const ThirtyDays = 30 * (24 * time.Hour)

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Short:   "Manage access keys",
		Long:    "Manage access keys for machine identities to authenticate with Infra and call the API",
		Aliases: []string{"key"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newKeysListCmd())
	cmd.AddCommand(newKeysAddCmd())
	cmd.AddCommand(newKeysRemoveCmd())

	return cmd
}

type keyCreateOptions struct {
	TTL               time.Duration
	ExtensionDeadline time.Duration
}

func newKeysAddCmd() *cobra.Command {
	var options keyCreateOptions

	cmd := &cobra.Command{
		Use:   "add ACCESS_KEY_NAME MACHINE_NAME",
		Short: "Create an access key for authentication",
		Example: `
# Create an access key for the machine "bot" called "first-key" that expires in 12 hours and must be used every hour to remain valid
infra keys add first-key bot --ttl=12h --extension-deadline=1h
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyName := args[0]
			machineName := args[1]

			if strings.Contains(keyName, " ") {
				return fmt.Errorf("key name cannot contain spaces")
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			machine, err := GetIdentityFromName(client, machineName)
			if err != nil {
				return err
			}

			resp, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{
				IdentityID:        machine.ID,
				Name:              keyName,
				TTL:               api.Duration(options.TTL),
				ExtensionDeadline: api.Duration(options.ExtensionDeadline),
			})
			if err != nil {
				return err
			}

			fmt.Printf("key: %s \n", resp.AccessKey)
			return nil
		},
	}

	cmd.Flags().DurationVar(&options.TTL, "ttl", ThirtyDays, "The total time that an access key will be valid for")
	cmd.Flags().DurationVar(&options.ExtensionDeadline, "extension-deadline", ThirtyDays, "A specified deadline that an access key must be used within to remain valid")

	return cmd
}

func newKeysRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove ACCESS_KEY_NAME",
		Aliases: []string{"rm"},
		Short:   "Delete an access key",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			keys, err := client.ListAccessKeys(api.ListAccessKeysRequest{Name: args[0]})
			if err != nil {
				return err
			}

			if len(keys) == 0 {
				return fmt.Errorf("no access key found with this name")
			}

			if len(keys) != 1 {
				return fmt.Errorf("invalid access key response, there should only be one access key that matches a name, but multiple were found")
			}

			err = client.DeleteAccessKey(keys[0].ID)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

type keyListOptions struct {
	MachineName string
}

func newKeysListCmd() *cobra.Command {
	var options keyListOptions

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List access keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			var keys []api.AccessKey
			if options.MachineName != "" {
				machine, err := GetIdentityFromName(client, options.MachineName)
				if err != nil {
					return err
				}

				keys, err = client.ListAccessKeys(api.ListAccessKeysRequest{IdentityID: machine.ID})
				if err != nil {
					return err
				}
			} else {
				keys, err = client.ListAccessKeys(api.ListAccessKeysRequest{})
				if err != nil {
					return err
				}
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
					Created:           k.Created.Relative("never"),
					Expires:           k.Expires.Relative("never"),
					ExtensionDeadline: k.ExtensionDeadline.Relative("never"),
				})
			}

			if len(rows) > 0 {
				printTable(rows)
			} else {
				fmt.Println("No access keys found")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.MachineName, "machine", "m", "", "The name of a machine to list access keys for")

	return cmd
}
