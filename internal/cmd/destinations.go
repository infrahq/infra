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

			if len(rows) > 0 {
				printTable(rows, cli.Stdout)
			} else {
				cli.Output("No destinations connected")
			}

			return nil
		},
	}
}

func newDestinationsRemoveCmd(cli *CLI) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "remove DESTINATION",
		Aliases: []string{"rm"},
		Short:   "Disconnect a destination",
		Example: "$ infra destinations remove docker-desktop",
		Args:    ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			forbiddenMsg := "You do not have privileges to disconnect destinations from infra; contact your admin\n\nRun `infra info` for more information about your session"
			name := args[0]
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			logging.S.Debugf("call server: list destinations named %q", name)
			destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: name})
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					return Error{
						Message: forbiddenMsg,
					}
				}
				return err
			}

			if destinations.Count == 0 && !force {
				return Error{Message: fmt.Sprintf("Destination %q not connected", name)}
			}

			logging.S.Debugf("deleting %s destinations named %q...", destinations.Count, name)
			for _, d := range destinations.Items {
				logging.S.Debugf("...call server: delete destination %s", d.ID)
				err := client.DeleteDestination(d.ID)
				if err != nil {
					if api.ErrorStatusCode(err) == 403 {
						return Error{
							Message: forbiddenMsg,
						}
					}
					return err
				}

				cli.Output("Disconnected destination %q from infra", d.Name)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Exit successfully even if destination does not exist")

	return cmd
}
