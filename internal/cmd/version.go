package cmd

import (
	"os"
	"fmt"
	"text/tabwriter"
	"context"

	"github.com/infrahq/infra/internal"
)

func version() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Client:\t", internal.Version)

	client, err := apiClientFromConfig()
	if err != nil {
		fmt.Fprintln(w, blue("✕")+" Could not retrieve client version")
		return err
	}

	// Note that we use the client to get this version, but it is in fact the server version
	res, _, err := client.VersionApi.Version(context.Background()).Execute()
	if err != nil {
		fmt.Fprintln(w, "Registry:\t", "not connected")
		return err
	}

	fmt.Fprintln(w, "Registry:\t", res.Version)
	fmt.Fprintln(w)

	return nil
}
