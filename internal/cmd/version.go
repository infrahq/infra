package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/mod/semver"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
)

type VersionOptions struct {
	Client           bool
	Server           bool
	internal.Options `mapstructure:",squash"`
}

func version(options *VersionOptions) error {
	clientVersion := internal.Version
	serverVersion := "disconnected"

	client, err := apiClient()
	if err != nil {
		return err
	}

	version, err := client.GetVersion()
	if err == nil {
		serverVersion = version.Version
	}

	err = checkUpdate(clientVersion, serverVersion)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed checking for updates:", err.Error())
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	fmt.Fprintln(w)

	if !options.Server {
		fmt.Fprintln(w, "Client:\t", clientVersion)
	}

	if !options.Client {
		fmt.Fprintln(w, "Server:\t", serverVersion)
	}

	fmt.Fprintln(w)

	return nil
}

func checkUpdate(clientVersion, serverVersion string) error {
	latestSemVer := "nonexistent"
	clientSemVer := fmt.Sprintf("v%s", clientVersion)
	serverSemVer := fmt.Sprintf("v%s", serverVersion)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://releases.infrahq.com/infra/latest", nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("%s", res.Status)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err == nil {
		latestSemVer = strings.TrimSpace(string(body))
	}

	var latestVersion string

	_, err = fmt.Sscanf(latestSemVer, "v%s", &latestVersion)
	if err != nil {
		return err
	}

	if clientSemVer != "v0.0.0-development" && semver.Compare(latestSemVer, clientSemVer) > 0 {
		logging.S.Warnf("Infra CLI (%s) is out of date. Please update to %s.", clientVersion, latestVersion)
	}

	if serverSemVer != "v0.0.0-development" && semver.IsValid(serverSemVer) && semver.Compare(latestSemVer, serverSemVer) > 0 {
		logging.S.Warnf("Infra (%s) is out of date. Please update to %s.", serverVersion, latestVersion)
	}

	return nil
}
