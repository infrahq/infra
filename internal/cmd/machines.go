package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	survey "github.com/AlecAivazis/survey/v2"

	"github.com/infrahq/infra/internal/api"
)

type MachinesCreateOptions struct {
	Name        string
	Description string
	Permissions string
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
	var options MachinesCreateOptions

	cmd := &cobra.Command{
		Use:   "create [NAME] [DESCRIPTION] [PERMISSIONS]",
		Short: "Create a machine identity, e.x. a service that needs to access infrastructure",
		Args:  cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				options.Name = args[0]
			}

			if len(args) > 1 {
				options.Description = args[1]
			}

			if len(args) == 3 {
				options.Permissions = args[2]
			}

			return createMachine(&options)
		},
	}

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

func createMachine(options *MachinesCreateOptions) error {
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
