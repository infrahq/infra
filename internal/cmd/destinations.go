package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func newDestinationsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List connected destinations",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			for _, d := range destinations {
				rows = append(rows, row{
					Name: d.Name,
					URL:  d.Connection.URL,
				})
			}

			printTable(rows)

			return nil
		},
	}
}

func newDestinationsAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add TYPE NAME",
		Short: "Connect a destination",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "kubernetes" {
				return fmt.Errorf("Supported types: `kubernetes`")
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			destination := &api.CreateMachineRequest{
				Name:        args[1],
				Description: fmt.Sprintf("%s %s destination", args[1], args[0]),
			}

			created, err := client.CreateMachine(destination)
			if err != nil {
				return err
			}

			destinationGrant := &api.CreateGrantRequest{
				Identity:  uid.NewMachinePolymorphicID(created.ID),
				Privilege: models.InfraConnectorRole,
				Resource:  "infra",
			}

			_, err = client.CreateGrant(destinationGrant)
			if err != nil {
				return err
			}

			lifetime := time.Hour * 24 * 365
			accessKey, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{
				MachineID: created.ID,
				Name:      fmt.Sprintf("access key presented by %s %s destination", args[1], args[0]),
				TTL:       lifetime.String(),
			})
			if err != nil {
				return err
			}

			config, err := currentHostConfig()
			if err != nil {
				return err
			}

			var sb strings.Builder
			sb.WriteString("    helm install infra-connector infrahq/infra")

			if len(args) > 1 {
				fmt.Fprintf(&sb, " --set connector.config.name=%s", args[1])
			}

			fmt.Fprintf(&sb, " --set connector.config.accessKey=%s", accessKey.AccessKey)
			fmt.Fprintf(&sb, " --set connector.config.server=%s", config.Host)

			// TODO: replace me with a certificate fingerprint
			// so even when users have self-signed certificates
			// infra can establish a secure TLS connection
			if config.SkipTLSVerify {
				sb.WriteString(" --set connector.config.skipTLSVerify=true")
			}

			fmt.Println()
			fmt.Println("Run the following command to connect a kubernetes cluster:")
			fmt.Println()
			fmt.Println(sb.String())
			fmt.Println()
			return nil
		},
	}
}

func newDestinationsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove DESTINATION",
		Aliases: []string{"rm"},
		Short:   "Disconnect a destination",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: args[0]})
			if err != nil {
				return err
			}

			if len(destinations) == 0 {
				return fmt.Errorf("no destinations named %s", args[0])
			}

			for _, d := range destinations {
				err := client.DeleteDestination(d.ID)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
