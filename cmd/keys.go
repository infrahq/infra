package cmd

import (
	"fmt"

	"github.com/infrahq/infra/internal/api"
	"github.com/spf13/cobra"
)

type keyCreateOptions struct {
	TTL               string `mapstructure:"ttl"`
	ExtensionDeadline string `mapstructure:"extension-deadline"`
}

func newKeysCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create ACCESS_KEY_NAME MACHINE_NAME",
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

			resp, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{MachineID: machine.ID, Name: keyName, TTL: options.TTL, ExtensionDeadline: options.ExtensionDeadline})
			if err != nil {
				return err
			}

			fmt.Printf("key: %s \n", resp.AccessKey)

			return nil
		},
	}

	cmd.Flags().StringP("ttl", "t", "", "The total time that an access key will be valid for")
	cmd.Flags().StringP("extension-deadline", "e", "", "A specified deadline that an access key must be used within to remain valid")

	return cmd
}

func newKeysDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete ACCESS_KEY_NAME",
		Short: "Delete access keys",
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
					ExtensionDeadline: k.ExtensionDeadline.String(),
				})
			}

			printTable(rows)

			return nil
		},
	}

	cmd.Flags().StringP("machine", "m", "", "The name of a machine to list access keys for")

	return cmd
}
