package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
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
	cmd.AddCommand(newDestinationsRemoveCmd())

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

			logging.S.Debug("call server: list destinations")
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

			logging.S.Debug("print destinations")
			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No destinations found.")
			}

			return nil
		},
	}
}

func newDestinationsRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
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

			logging.S.Debug("call server: list destinations named [%s]", args[0])
			destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: args[0]})
			if err != nil {
				return err
			}

			if destinations.Count == 0 {
				if force {
					return nil
				}
				return Error{
					Message: fmt.Sprintf("No destinations named [%s].", args[0]),
				}
			}

			logging.S.Debug("deleting %s destinations named [%s]...", destinations.Count, args[0])
			for _, d := range destinations.Items {
				logging.S.Debug("...call server: delete destination [%s]", d.ID)
				err := client.DeleteDestination(d.ID)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Exit successfully when destination not found")

	return cmd
}
