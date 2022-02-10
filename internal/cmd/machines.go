package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/api"
)

type machineOptions struct {
	Description string `mapstructure:"description"`
	Permissions string `mapstructure:"permissions"`
}

func newMachinesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [NAME]",
		Short: "Create a machine identity, e.g. a service that needs to access infrastructure",
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
	cmd.Flags().StringP("permissions", "p", "", "Permissions of the machine identity")

	return cmd
}

func newMachinesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List machines",
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
				Name        string   `header:"Name"`
				Permissions []string `header:"Permissions"`
				Description string   `header:"Description"`
			}

			var rows []row
			for _, m := range machines {
				rows = append(rows, row{
					Name:        m.Name,
					Permissions: m.Permissions,
					Description: m.Description,
				})
			}

			printTable(rows)

			return nil
		},
	}
}

func newMachinesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove MACHINE",
		Short: "Remove a machine identity",
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

	_, err = client.CreateMachine(&api.CreateMachineRequest{Name: name, Description: options.Description, Permissions: strings.Split(options.Permissions, " ")})
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
