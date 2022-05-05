package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

const ThirtyDays = 30 * (24 * time.Hour)

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
	cmd.AddCommand(newKeysRemoveCmd())

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
		Use:   "add IDENTITY",
		Short: "Create an access key",
		Long:  `Create an access key for an identity.`,
		Example: `
# Create an access key named 'example-key' that expires in 12 hours
$ infra keys add example-key identity@example.com --ttl=12h
`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identityName := args[0]

			if options.Name != "" {
				if strings.Contains(options.Name, " ") {
					return fmt.Errorf("key name cannot contain spaces")
				}
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			identity, err := GetIdentityByName(client, identityName)
			if err != nil {
				return err
			}

			resp, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{
				IdentityID:        identity.ID,
				Name:              options.Name,
				TTL:               api.Duration(options.TTL),
				ExtensionDeadline: api.Duration(options.ExtensionDeadline),
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cli.Stderr, "Created access key for %q\n", identityName)
			cli.Output("Name: %s", resp.Name)
			cli.Output("Key: %s", resp.AccessKey)
			return nil
		},
	}

	cmd.Flags().StringVar(&options.Name, "name", "", "The name of the access key")
	cmd.Flags().DurationVar(&options.TTL, "ttl", ThirtyDays, "The total time that the access key will be valid for")
	cmd.Flags().DurationVar(&options.ExtensionDeadline, "extension-deadline", ThirtyDays, "A specified deadline that the access key must be used within to remain valid")

	return cmd
}

func newKeysRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove KEY",
		Aliases: []string{"rm"},
		Short:   "Delete an access key",
		Args:    ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			keys, err := client.ListAccessKeys(api.ListAccessKeysRequest{Name: args[0]})
			if err != nil {
				return err
			}

			if keys.Count == 0 {
				return fmt.Errorf("no access key found with this name")
			}

			if keys.Count != 1 {
				return fmt.Errorf("invalid access key response, there should only be one access key that matches a name, but multiple were found")
			}

			key := keys.Items[0]

			err = client.DeleteAccessKey(key.ID)
			if err != nil {
				return err
			}

			issuedFor := key.IssuedForName
			if issuedFor == "" {
				issuedFor = key.IssuedFor.String()
			}

			fmt.Fprintf(os.Stderr, "Deleted access key %q issued for %q\n", key.Name, issuedFor)

			return nil
		},
	}
}

type keyListOptions struct {
	IdentityName string
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

			keys := &api.ListResponse[api.AccessKey]{}
			if options.IdentityName != "" {
				identity, err := GetIdentityByName(client, options.IdentityName)
				if err != nil {
					return err
				}

				keys, err = client.ListAccessKeys(api.ListAccessKeysRequest{IdentityID: identity.ID})
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
			for _, k := range keys.Items {
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
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No access keys found")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.IdentityName, "identity", "i", "", "The name of a identity to list access keys for")

	return cmd
}
