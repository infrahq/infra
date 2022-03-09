package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/api"
)

type machineOptions struct {
	Description string `mapstructure:"description"`
}

func newIdentitiesAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Create a machine identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			var options machineOptions
			if err := parseOptions(cmd, &options, "INFRA_MACHINE"); err != nil {
				return err
			}

			return createMachine(name, &options)
		},
	}

	cmd.Flags().StringP("description", "d", "", "Description of the machine identity")

	return cmd
}

func newIdentitiesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all identities (users & machines)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			machines, err := client.ListMachines(api.ListMachinesRequest{})
			if err != nil {
				return err
			}

			type row struct {
				Name        string `header:"Name"`
				Description string `header:"Description"`
			}

			var rows []row
			for _, m := range machines {
				rows = append(rows, row{
					Name:        m.Name,
					Description: m.Description,
				})
			}

			printTable(rows)

			return nil
		},
	}
}

func newIdentitiesRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove MACHINE",
		Short: "Delete a machine identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			machines, err := client.ListMachines(api.ListMachinesRequest{Name: args[0]})
			if err != nil {
				return err
			}

			for _, m := range machines {
				err := client.DeleteMachine(m.ID)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func createMachine(name string, options *machineOptions) error {
	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	_, err = client.CreateMachine(&api.CreateMachineRequest{Name: name, Description: options.Description})
	if err != nil {
		return err
	}

	return nil
}

func getMachineFromName(client *api.Client, name string) (*api.Machine, error) {
	machines, err := client.ListMachines(api.ListMachinesRequest{Name: name})
	if err != nil {
		return nil, err
	}

	if len(machines) == 0 {
		return nil, fmt.Errorf("no machine found with this name")
	}

	if len(machines) != 1 {
		return nil, fmt.Errorf("invalid machines response, there should only be one machine that matches a name, but multiple were found")
	}
	return &machines[0], nil
}
