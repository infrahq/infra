package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/infrahq/infra/api"
	humanfmt "github.com/infrahq/infra/internal/format"
	"github.com/infrahq/infra/internal/logging"
)

const (
	DestinationStatusConnected    = "Connected"
	DestinationStatusPending      = "Pending"
	DestinationStatusDisconnected = "Disconnected"
)

func newDestinationsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destinations",
		Aliases: []string{"dst", "dest", "destination"},
		Short:   "Manage destinations",
		GroupID: groupManagement,
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
	var format string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List connected destinations",
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			ctx := context.Background()

			logging.Debugf("call server: list destinations")
			destinations, err := listAll(ctx, client.ListDestinations, api.ListDestinationsRequest{})
			if err != nil {
				return err
			}

			switch format {
			case "json":
				jsonOutput, err := json.Marshal(destinations)
				if err != nil {
					return err
				}
				cli.Output(string(jsonOutput))
			case "yaml":
				yamlOutput, err := yaml.Marshal(destinations)
				if err != nil {
					return err
				}
				cli.Output(string(yamlOutput))
			default:
				type row struct {
					Name     string `header:"NAME"`
					Kind     string `header:"KIND"`
					URL      string `header:"URL"`
					Status   string `header:"STATUS"`
					LastSeen string `header:"LAST SEEN"`
				}

				var rows []row
				for _, d := range destinations {
					var status string
					switch {
					case !d.Connected:
						status = DestinationStatusDisconnected
					case d.Connection.URL == "":
						status = DestinationStatusPending
					default:
						status = DestinationStatusConnected
					}

					rows = append(rows, row{
						Name:     d.Name,
						Kind:     d.Kind,
						URL:      d.Connection.URL,
						Status:   status,
						LastSeen: humanfmt.HumanTime(d.LastSeen.Time(), "never"),
					})
				}
				if len(rows) > 0 {
					printTable(rows, cli.Stdout)
				} else {
					cli.Output("No destinations connected")
				}
			}
			return nil
		},
	}

	addFormatFlag(cmd.Flags(), &format)
	return cmd
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
			name := args[0]
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			ctx := context.Background()

			logging.Debugf("call server: list destinations named %q", name)
			destinations, err := client.ListDestinations(ctx, api.ListDestinationsRequest{Name: name})
			if err != nil {
				if api.ErrorStatusCode(err) == 403 {
					logging.Debugf("%s", err.Error())
					return Error{
						Message: "Cannot disconnect destination: missing privileges for ListDestinations",
					}
				}
				return err
			}

			if destinations.Count == 0 && !force {
				return Error{Message: fmt.Sprintf("Destination %q not connected", name)}
			}

			logging.Debugf("deleting %d destinations named %q...", destinations.Count, name)
			for _, d := range destinations.Items {
				logging.Debugf("...call server: delete destination %s", d.ID)
				err := client.DeleteDestination(ctx, d.ID)
				if err != nil {
					if api.ErrorStatusCode(err) == 403 {
						logging.Debugf("%s", err.Error())
						return Error{
							Message: "Cannot disconnect destination: missing privileges for DeleteDestination",
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
