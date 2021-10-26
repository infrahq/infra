package cmd

import (
	"os"
	"path/filepath"

	"github.com/infrahq/infra/internal/api"
)

func logout(host string) error {
	config, err := readHostConfig(host)
	if err != nil {
		return err
	}

	if config == nil {
		return nil
	}

	client, err := NewAPIClient(config.Host, config.SkipTLSVerify)
	if err == nil {
		_, _ = client.AuthAPI.Logout(NewAPIContext(config.Token)).Execute()
	}

	// only clean up cache and destinations if logging out of current host
	if config.Current {
		_ = updateKubeconfig(api.User{})

		infraDir, err := infraHomeDir()
		if err == nil {
			os.RemoveAll(filepath.Join(infraDir, "cache"))
			os.Remove(filepath.Join(infraDir, "destinations"))
		}
	}

	_ = removeHostConfig(config.Host)

	return nil
}
