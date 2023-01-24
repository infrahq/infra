package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
)

func newUseCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "use DESTINATION",
		Short: "Access a destination",
		Example: `
# Use a Kubernetes context
$ infra use development

# Use a Kubernetes namespace context
$ infra use development.kube-system`,
		Args:              ExactArgs(1),
		GroupID:           groupCore,
		ValidArgsFunction: getUseCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			destination := args[0]

			client, err := cli.apiClient()
			if err != nil {
				return err
			}

			err = updateKubeconfig(client)
			if err != nil {
				return err
			}

			parts := strings.Split(destination, ".")

			if len(parts) == 1 {
				return kubernetesSetContext(cli, destination, "")
			}

			return kubernetesSetContext(cli, parts[0], parts[1])
		},
	}
}

func getUseCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	opts, err := defaultClientOpts()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	client, err := NewAPIClient(opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	_, destinations, grants, err := getUserDestinationGrants(client, "")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	resources := make(map[string]struct{}, len(grants))

	for _, g := range grants {
		resource := api.FormatResourceURN(g.DestinationName, g.DestinationResource)
		resources[resource] = struct{}{}
	}

	validArgs := make([]string, 0, len(resources))

	for r := range resources {
		var exists bool
		for _, d := range destinations {
			if strings.HasPrefix(r, d.Name) {
				exists = true
				break
			}
		}

		if exists {
			validArgs = append(validArgs, r)
		}

	}

	return validArgs, cobra.ShellCompDirectiveNoSpace
}
