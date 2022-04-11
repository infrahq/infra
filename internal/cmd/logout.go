package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/logging"
)

func newLogoutCmd() *cobra.Command {
	var purge bool

	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "Log out of Infra",
		Example: "$ infra logout",
		Group:   "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logout(purge)
		},
	}

	cmd.Flags().BoolVar(&purge, "purge", false, "remove Infra host from config")

	return cmd
}

func logout(purge bool) error {
	config, err := readConfig()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return nil
		}

		return err
	}

	for i, hostConfig := range config.Hosts {
		config.Hosts[i].AccessKey = ""

		client, err := apiClient(hostConfig.Host, hostConfig.AccessKey, hostConfig.SkipTLSVerify)
		if err != nil {
			logging.S.Warn(err.Error())
			continue
		}

		_ = client.Logout()
	}

	if purge {
		config.Hosts = nil
	}

	if err := clearKubeconfig(); err != nil {
		return err
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	return nil
}
