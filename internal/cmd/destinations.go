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
		Use:   "add DESTINATION",
		Short: "Connect an infrastructure destination to Infra",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(args[0], ".")
			if len(parts) != 2 {
				return fmt.Errorf("invalid input for destination: expected \"<TYPE>.<NAME>\", got %q", args[0])
			}

			supportedTypes := []string{
				"kubernetes",
			}

			supportedType := false
			for _, t := range supportedTypes {
				if parts[0] == t {
					supportedType = true
					break
				}
			}

			if !supportedType {
				return fmt.Errorf("unknown destination type: %q. supported types: %v", parts[0], supportedTypes)
			}

			destination := &api.CreateMachineRequest{
				Name:        parts[1],
				Description: fmt.Sprintf("%s destination", args[0]),
			}

			client, err := defaultAPIClient()
			if err != nil {
				return err
			}

			created, err := client.CreateMachine(destination)
			if err != nil {
				return err
			}

			destinationGrant := &api.CreateGrantRequest{
				Subject:   uid.NewMachinePolymorphicID(created.ID),
				Privilege: models.InfraConnectorRole,
				Resource:  "infra",
			}

			_, err = client.CreateGrant(destinationGrant)
			if err != nil {
				return err
			}

			lifetime := time.Hour * 24 * 365
			extensionDeadline := time.Hour * 24
			accessKey, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{
				MachineID:         created.ID,
				Name:              fmt.Sprintf("%s destination access key", args[0]),
				TTL:               api.Duration(lifetime),
				ExtensionDeadline: api.Duration(extensionDeadline),
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

			fmt.Fprintf(&sb, " --set connector.config.name=%s", parts[1])
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
