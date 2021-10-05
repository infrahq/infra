package cmd

import (
	"os"
	"path/filepath"

	"github.com/infrahq/infra/internal/api"
)

func logout() error {
	config, err := readCurrentConfig()
	if err == nil {
		client, err := NewApiClient(config.Host, config.SkipTLSVerify)
		if err == nil {
			_, _ = client.AuthApi.Logout(NewApiContext(config.Token)).Execute()
		}
	}

	_ = updateKubeconfig([]api.Destination{})

	homeDir, err := os.UserHomeDir()
	if err == nil {
		os.RemoveAll(filepath.Join(homeDir, ".infra", "cache"))
		os.Remove(filepath.Join(homeDir, ".infra", "destinations"))
	}

	_ = removeConfig()

	return nil
}
