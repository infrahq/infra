package cmd

import (
	"os"
	"path/filepath"

	"github.com/infrahq/infra/internal/api"
)

func logout(registry string) error {
	config, err := readRegistryConfig(registry)
	if err != nil {
		return err
	}

	if config == nil {
		return nil
	}

	client, err := NewApiClient(config.Host, config.SkipTLSVerify)
	if err == nil {
		_, _ = client.AuthApi.Logout(NewApiContext(config.Token)).Execute()
	}

	// only clean up cache and destinations if logging out of current registry
	if config.Current {
		_ = updateKubeconfig(api.User{})

		infraDir, err := infraHomeDir()
		if err == nil {
			os.RemoveAll(filepath.Join(infraDir, "cache"))
			os.Remove(filepath.Join(infraDir, "destinations"))
		}
	}

	_ = removeRegistryConfig(config.Host)

	return nil
}
