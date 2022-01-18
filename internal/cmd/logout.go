package cmd

import (
	"errors"

	"github.com/infrahq/infra/internal/logging"
)

func logout() error {
	config, err := readConfig()
	if errors.Is(err, ErrConfigNotFound) {
		logging.S.Debug(err.Error())
		return nil
	}

	if err != nil {
		logging.S.Debug(err.Error())
		return err
	}

	for _, hostConfig := range config.Hosts {
		if err := removeHostConfig(hostConfig.Host); err != nil {
			logging.S.Warn(err.Error())
			continue
		}

		client, err := apiClient(hostConfig.Host, hostConfig.Token, hostConfig.SkipTLSVerify)
		if err != nil {
			logging.S.Warn(err.Error())
			continue
		}

		client.Logout()
	}

	return clearKubeconfig()
}
