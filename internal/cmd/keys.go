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
		Aliases: []string{"key"},
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newKeysListCmd())
	cmd.AddCommand(newKeysAddCmd())
	cmd.AddCommand(newKeysRemoveCmd())

	return cmd
}

type keyCreateOptions struct {
	TTL               string `mapstructure:"ttl"`
	ExtensionDeadline string `mapstructure:"extension-deadline"`
}

func newKeysAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add KEY IDENTITY",
		Short: "Create an access key",
		Long:  `Create an access key. Only machine identities are supported at this time.`,
		Example: `
# Create an access key named 'key1' that expires in 12 hrs
$ infra keys add key1 machineA --ttl=12h
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options keyCreateOptions
			if err := parseOptions(cmd, &options, "INFRA_KEYS"); err != nil {
				return err
			}

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

			deadline := ThirtyDays
			if options.ExtensionDeadline != "" {
				deadline, err = time.ParseDuration(options.ExtensionDeadline)
				if err != nil {
					return err
				}
			}

			ttl := ThirtyDays
			if options.TTL != "" {
				ttl, err = time.ParseDuration(options.TTL)
				if err != nil {
					return fmt.Errorf("parsing ttl: %w", err)
				}
			}

			resp, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{IdentityID: machine.ID, Name: keyName, TTL: api.Duration(ttl), ExtensionDeadline: api.Duration(deadline)})
			if err != nil {
				return err
			}

			fmt.Printf("key: %s \n", resp.AccessKey)

			return nil
		},
	}

	cmd.Flags().String("ttl", "", "The total time that an access key will be valid for, defaults to 30 days")
	cmd.Flags().String("extension-deadline", "", "A specified deadline that an access key must be used within to remain valid, defaults to 30 days")

	return cmd
}

func newKeysRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove KEY",
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
	MachineName string `mapstructure:"machine"`
}

func newKeysListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List access keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options keyListOptions
			if err := parseOptions(cmd, &options, "INFRA_KEYS"); err != nil {
				return err
			}

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
				rows = append(rows, row{
					Name:              k.Name,
					IssuedFor:         k.IssuedFor.String(),
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

	cmd.Flags().StringP("machine", "m", "", "The name of a machine to list access keys for")

	return cmd
}
