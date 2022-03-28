package cmd

import (
	"errors"

	"github.com/infrahq/infra/internal/logging"
)

func logout(purge bool) error {
	config, err := readConfig()
	if err != nil {
		logging.S.Debug(err.Error())

		if errors.Is(err, ErrConfigNotFound) {
			return nil
		}

		return err
	}

	for _, hostConfig := range config.Hosts {
		if err := removeHostConfig(hostConfig.Host, purge); err != nil {
			logging.S.Warn(err.Error())
			continue
		}

		client, err := apiClient(hostConfig.Host, hostConfig.AccessKey, hostConfig.SkipTLSVerify)
		if err != nil {
			logging.S.Warn(err.Error())
			continue
		}

		if err := client.Logout(); err != nil {
			logging.S.Warnf("failed to logout: %v", err)
		}
	}

	return clearKubeconfig()
}
