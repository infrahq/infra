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

	for i, hostConfig := range config.Hosts {
		if !purge {
			config.Hosts[i].AccessKey = ""
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

	if purge {
		config.Hosts = nil
	}
	if err := writeConfig(config); err != nil {
		logging.S.Warnf("failed to write client host config: %v", err)
	}
	return clearKubeconfig()
}
