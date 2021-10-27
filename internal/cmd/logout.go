package cmd

import (
	"os"
	"path/filepath"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
)

type LogoutOptions struct {
	internal.GlobalOptions
}

func cleanupKubeconfig(config *ClientHostConfig) error {
	if config.Current {
		_ = updateKubeconfig(api.User{})

		infraDir, err := infraHomeDir()
		if err == nil {
			os.RemoveAll(filepath.Join(infraDir, "cache"))
		}
	}

	return removeHostConfig(config.Host)
}

func logoutOne(config *ClientHostConfig) error {
	client, err := apiClientFromConfig(config.Host)
	if err != nil {
		logging.S.Warnf("%s", err.Error())
		return cleanupKubeconfig(config)
	}

	ctx, err := apiContextFromConfig(config.Host)
	if err != nil {
		logging.S.Warnf("%s", err.Error())
		return cleanupKubeconfig(config)
	}

	_, err = client.AuthAPI.Logout(ctx).Execute()
	if err != nil {
		logging.S.Warnf("%s", err.Error())
		return cleanupKubeconfig(config)
	}

	return cleanupKubeconfig(config)
}

func logout(options *LogoutOptions) error {
	if options.Host == "" {
		configs, _ := readConfig()
		for i := range configs.Hosts {
			_ = logoutOne(&configs.Hosts[i])
		}

		return nil
	}

	config, err := readHostConfig(options.Host)
	if err != nil {
		logging.S.Warnf("%s", err.Error())
		return nil
	}

	_ = logoutOne(config)

	return nil
}
