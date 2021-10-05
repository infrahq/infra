package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infrahq/infra/internal/api"
)

func logout() error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	if config.Token == "" {
		return nil
	}

	client, err := NewApiClient(config.Host, config.SkipTLSVerify)
	if err != nil {
		return err
	}

	_, err = client.AuthApi.Logout(NewApiContext(config.Token)).Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	err = removeConfig()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	os.Remove(filepath.Join(homeDir, ".infra", "destinations"))

	return updateKubeconfig([]api.Destination{})
}
