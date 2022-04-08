package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
)

const ThirtyDays = 30 * (24 * time.Hour)

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Short:   "Manage access keys",
		Long:    "Manage access keys for machine identities to authenticate with Infra and call the API",
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
		Use:   "add ACCESS_KEY_NAME MACHINE_NAME",
		Short: "Create an access key for authentication",
		Example: `
# Create an access key for the machine "bot" called "first-key" that expires in 12 hours and must be used every hour to remain valid
infra keys add first-key bot --ttl=12h --extension-deadline=1h
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

			infraProvider, err := GetProviderByName(client, models.InternalInfraProviderName)
			if err != nil {
				logging.S.Debug(err)
				return fmt.Errorf("no infra provider found, to manage local users create a local provider named 'infra'")
			}

			machine, err := GetIdentityFromName(client, machineName, infraProvider.ID)
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
		Use:   "remove ACCESS_KEY_NAME",
		Short: "Delete an access key",
		Args:  cobra.ExactArgs(1),
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
				infraProvider, err := GetProviderByName(client, models.InternalInfraProviderName)
				if err != nil {
					logging.S.Debug(err)
					return fmt.Errorf("no infra provider found, to manage local users create a local provider named 'infra'")
				}

				machine, err := GetIdentityFromName(client, options.MachineName, infraProvider.ID)
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
				ID                string `header:"ID"`
				Name              string `header:"NAME"`
				IssuedFor         string `header:"ISSUED FOR"`
				Created           string `header:"CREATED"`
				Expires           string `header:"EXPIRES"`
				ExtensionDeadline string `header:"EXTENSION DEADLINE"`
			}

			var rows []row
			for _, k := range keys {
				rows = append(rows, row{
					ID:                k.ID.String(),
					Name:              k.Name,
					IssuedFor:         k.IssuedFor.String(),
					Created:           k.Created.String(),
					Expires:           k.Expires.String(),
					ExtensionDeadline: k.ExtensionDeadline.Format(time.RFC3339),
				})
			}

			printTable(rows)

			return nil
		},
	}

	cmd.Flags().StringP("machine", "m", "", "The name of a machine to list access keys for")

	return cmd
}
