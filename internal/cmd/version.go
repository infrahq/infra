package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
)

func version() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Client:\t", internal.Version)

	// Note that we use the client to get this version, but it is in fact the server version
	client, err := defaultAPIClient()
	if err != nil {
		fmt.Fprintln(w, "Server:\t", "disconnected")
		logging.S.Debug(err)
		fmt.Fprintln(w)
		return nil
	}

	version, err := client.GetVersion()
	if err != nil {
		fmt.Fprintln(w, "Server:\t", "disconnected")
		logging.S.Debug(err)
		fmt.Fprintln(w)
		return nil
	}

	fmt.Fprintln(w, "Server:\t", version.Version)
	fmt.Fprintln(w)

	return nil
}
