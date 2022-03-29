package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

type keyCreateOptions struct {
	TTL               string `mapstructure:"ttl"`
	ExtensionDeadline string `mapstructure:"extension-deadline"`
}

func newKeysAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add ACCESS_KEY_NAME MACHINE_NAME",
		Short: "Create an access key for authentication",
		Example: `
# Create an access key for the machine "wall-e" called main that expires in 12 hours and must be used every hour to remain valid
infra keys create main wall-e 12h --extension-deadline=1h
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options keyCreateOptions
			if err := parseOptions(cmd, &options, "INFRA_KEYS"); err != nil {
				return err
			}

			keyName := args[0]
			machineName := args[1]

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			machine, err := getMachineFromName(client, machineName)
			if err != nil {
				return err
			}

			deadline := 1 * time.Hour
			if options.ExtensionDeadline != "" {
				deadline, err = time.ParseDuration(options.ExtensionDeadline)
				if err != nil {
					return err
				}
			}

			ttl := 24 * time.Hour
			if options.TTL != "" {
				ttl, err = time.ParseDuration(options.TTL)
				if err != nil {
					return fmt.Errorf("parsing ttl: %w", err)
				}
			}

			resp, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{MachineID: machine.ID, Name: keyName, TTL: api.Duration(ttl), ExtensionDeadline: api.Duration(deadline)})
			if err != nil {
				return err
			}

			fmt.Printf("key: %s \n", resp.AccessKey)

			return nil
		},
	}

	cmd.Flags().String("ttl", "", "The total time that an access key will be valid for")
	cmd.Flags().String("extension-deadline", "", "A specified deadline that an access key must be used within to remain valid")

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
				machine, err := getMachineFromName(client, options.MachineName)
				if err != nil {
					return err
				}

				keys, err = client.ListAccessKeys(api.ListAccessKeysRequest{MachineID: machine.ID})
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
					IssuedFor:         string(k.IssuedFor),
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
