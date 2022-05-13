package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
)

func newVersionCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the Infra version",
		Group: "Other commands:",
		Args:  NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return version(cli)
		},
	}
}

func version(cli *CLI) error {
	w := tabwriter.NewWriter(cli.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Client:\t", strings.TrimPrefix(api.GetClientVersion(), "v"))

	// Note that we use the client to get this version, but it is in fact the server version
	client, err := defaultAPIClient()
	if err != nil {
		fmt.Fprintln(w, "Server:\t", "disconnected")
		logging.S.Debug(err)
		fmt.Fprintln(w)

		return nil
	}

	version, err := client.GetServerVersion()
	if err != nil {
		fmt.Fprintln(w, "Server:\t", "disconnected")
		logging.S.Debug(err)
		fmt.Fprintln(w)

		return nil
	}

	fmt.Fprintln(w, "Server:\t", strings.TrimPrefix(version.Version, "v"))
	fmt.Fprintln(w)

	return nil
}
