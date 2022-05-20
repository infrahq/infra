package cmd

import (
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

func newDestinationsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destinations",
		Aliases: []string{"dst", "dest", "destination"},
		Short:   "Manage destinations",
		Group:   "Management commands:",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := rootPreRun(cmd.Flags()); err != nil {
				return err
			}
			return mustBeLoggedIn()
		},
	}

	cmd.AddCommand(newDestinationsListCmd(cli))
	cmd.AddCommand(newDestinationsRemoveCmd(cli))

	return cmd
}

func newDestinationsListCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List connected destinations",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			destinations, err := client.ListDestinations(api.ListDestinationsRequest{})
			if err != nil {
				return err
			}

			type row struct {
				Name string `header:"NAME"`
				URL  string `header:"URL"`
			}

			var rows []row
			for _, d := range destinations.Items {
				rows = append(rows, row{
					Name: d.Name,
					URL:  d.Connection.URL,
				})
			}

			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No destinations found")
			}

			return nil
		},
	}
}

func newDestinationsRemoveCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "remove DESTINATION",
		Aliases: []string{"rm"},
		Short:   "Disconnect a destination",
		Example: "$ infra destinations remove docker-desktop",
		Args:    ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: args[0]})
			if err != nil {
				return err
			}

			for _, d := range destinations.Items {
				err := client.DeleteDestination(d.ID)
				if err != nil {
					return err
				}

				cli.Output("Removed destination %q", d.Name)
			}

			return nil
		},
	}
}
