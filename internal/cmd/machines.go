package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	survey "github.com/AlecAivazis/survey/v2"

	"github.com/infrahq/infra/internal/api"
)

type machineOptions struct {
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
	Permissions string `mapstructure:"permissions"`
}

var (
	surveyName = &survey.Question{
		Name:     "name",
		Prompt:   &survey.Input{Message: "Name: "},
		Validate: survey.Required,
	}
	surveyDescription = &survey.Question{
		Name:     "description",
		Prompt:   &survey.Input{Message: "Description: "},
		Validate: survey.Required,
	}
	surveyPermissions = &survey.Question{
		Name:     "permissions",
		Prompt:   &survey.Input{Message: "Permissions: "},
		Validate: survey.Required,
	}
)

func newMachinesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a machine identity, e.g. a service that needs to access infrastructure",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options machineOptions
			if err := parseOptions(cmd, &options, "INFRA_MACHINE"); err != nil {
				return err
			}

			return createMachine(&options)
		},
	}

	cmd.Flags().StringP("name", "n", "", "Name of the machine identity")
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
		Args: cobra.ExactArgs(1),
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

func createMachine(options *machineOptions) error {
	requiredInformation := []*survey.Question{}

	if options.Name == "" {
		requiredInformation = append(requiredInformation, surveyName)
	}
	if options.Description == "" {
		requiredInformation = append(requiredInformation, surveyDescription)
	}
	if options.Permissions == "" {
		requiredInformation = append(requiredInformation, surveyPermissions)
	}

	if len(requiredInformation) > 0 {
		err := survey.Ask(requiredInformation, options)
		if err != nil {
			return err
		}
	}

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	_, err = client.CreateMachine(&api.CreateMachineRequest{Name: options.Name, Description: options.Description, Permissions: strings.Split(options.Permissions, " ")})
	if err != nil {
		return err
	}

	return nil
}
