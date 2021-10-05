package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/infrahq/infra/internal"
	"golang.org/x/mod/semver"
)

type VersionOptions struct {
	Client   bool
	Registry bool
}

func version(options VersionOptions) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	clientVersion := internal.Version
	serverVersion := "not connected"

	// Note that we use the client to get this version, but it is in fact the server version
	client, err := apiClientFromConfig()
	if err == nil {
		res, _, err := client.VersionApi.Version(context.Background()).Execute()
		if err == nil {
			serverVersion = res.Version
		}
	}

	clientSemVer := fmt.Sprintf("v%s", serverVersion)
	serverSemVer := fmt.Sprintf("v%s", clientVersion)

	if semver.Compare(clientSemVer, serverSemVer) > 0 {
		fmt.Fprintf(w, "ERROR: Your client (%s) is out of date. Please update to %s.\n", clientVersion, serverVersion)
	}

	fmt.Fprintln(w)

	if !options.Registry {
		fmt.Fprintln(w, "Client:\t", clientVersion)
	}

	if !options.Client {
		fmt.Fprintln(w, "Registry:\t", serverVersion)
	}

	fmt.Fprintln(w)

	return nil
}
