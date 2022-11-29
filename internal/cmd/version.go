package cmd

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
)

func newVersionCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Display the Infra version",
		GroupID: groupOther,
		Args:    NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return version(cli)
		},
	}
}

func version(cli *CLI) error {
	ctx := context.Background()

	w := tabwriter.NewWriter(cli.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Client:\t", strings.TrimPrefix(internal.FullVersion(), "v"))

	// Note that we use the client to get this version, but it is in fact the server version
	client, err := cli.apiClient()
	if err != nil {
		fmt.Fprintln(w, "Server:\t", "disconnected")
		logging.Debugf("%s", err.Error())
		fmt.Fprintln(w)

		return nil
	}

	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	// Don't bother printing the server version for SaaS
	if strings.HasSuffix(config.Host, ".infrahq.com") {
		fmt.Fprintln(w)
		return nil
	}

	version, err := client.GetServerVersion(ctx)
	if err != nil {
		fmt.Fprintln(w, "Server:\t", "disconnected")
		logging.Debugf("%s", err.Error())
		fmt.Fprintln(w)

		return nil
	}

	fmt.Fprintln(w, "Server:\t", strings.TrimPrefix(version.Version, "v"))
	fmt.Fprintln(w)

	return nil
}
