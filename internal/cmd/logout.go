package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/internal/logging"
)

func newLogoutCmd() *cobra.Command {
	var (
		clear bool
		url   string
		all   bool
	)

	cmd := &cobra.Command{
		Use:     "logout [URL]",
		Short:   "Log out of Infra",
		Example: "$ infra logout",
		Args:    cobra.MaximumNArgs(1),
		Group:   "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if all {
					fmt.Fprintf(os.Stderr, "Server is specified. Ignoring flag [--all] and logging out of server %s", args[1])
				}
				url = args[0]
			}
			return logout(clear, url, all)
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "clear from list of servers")
	cmd.Flags().BoolVar(&all, "all", false, "logout of all servers")

	return cmd
}

func logoutOfServer(hostConfig ClientHostConfig) {
	client, err := apiClient(hostConfig.Host, hostConfig.AccessKey, hostConfig.SkipTLSVerify)
	if err != nil {
		logging.S.Warn(err.Error())
	}

	_ = client.Logout()

}

func logout(clear bool, url string, all bool) error {
	config, err := readConfig()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return nil
		}

		return err
	}

	// Log out of server(s)
	for i, hostConfig := range config.Hosts {
		if all || url == hostConfig.Host || url == "" && hostConfig.Current {
			logoutOfServer(hostConfig)

			config.Hosts[i].AccessKey = ""
			config.Hosts[i].PolymorphicID = ""
			config.Hosts[i].Name = ""
		}
	}

	// Clear from list of saved servers
	var newHostConfigs []ClientHostConfig
	if clear {
		if all {
			config.Hosts = nil
		} else {
			for i := range config.Hosts {
				if url == "" && config.Hosts[i].Current {
					continue
				}

				if url != "" && url == config.Hosts[i].Name {
					continue
				}

				newHostConfigs = append(newHostConfigs, config.Hosts[i])
			}
		}
		config.Hosts = newHostConfigs
	}

	if err := clearKubeconfig(); err != nil {
		return err
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	return nil
}
