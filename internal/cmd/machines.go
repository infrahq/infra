package cmd

import (
	"strings"

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
