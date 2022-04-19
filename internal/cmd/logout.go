package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
)

func newLogoutCmd() *cobra.Command {
	var (
		clear  bool
		server string
		all    bool
	)

	cmd := &cobra.Command{
		Use:   "logout [SERVER]",
		Short: "Log out of Infra",
		Long: `Log out of Infra
Note: [SERVER] and [--all] cannot be both specified. Choose either one or all servers.`,
		Example: `# Log out of current server
$ infra logout
		
# Log out of a specific server
$ infra logout infraexampleserver.com
		
# Logout of all servers
$ infra logout --all 
		
# Log out of current server and clear from list 
$ infra logout --clear
		
# Log out of a specific server and clear from list
$ infra logout infraexampleserver.com --clear 
		
# Logout and clear list of all servers 
$ infra logout --all --clear`,
		Args:  cobra.MaximumNArgs(1),
		Group: "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if all {
					return fmt.Errorf("Argument [SERVER] and flag [--all] cannot be both specified.")
				}
				server = args[0]
			}
			logging.S.Debugf("flags set:")
			if clear {
				logging.S.Debug(" --clear")
			}
			if all {
				logging.S.Debug(" --all")
			}
			return logout(clear, server, all)
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "clear from list of servers")
	cmd.Flags().BoolVar(&all, "all", false, "logout of all servers")

	return cmd
}

func logoutOfServer(hostConfig *ClientHostConfig) (bool, error) {
	if !hostConfig.isLoggedIn() {
		logging.S.Debugf("requested but not logged in to server [%s]", hostConfig.Host)
		return false, nil
	}

	client, err := apiClient(hostConfig.Host, hostConfig.AccessKey, hostConfig.SkipTLSVerify)
	if err != nil {
		if !errors.Is(err, api.ErrUnauthorized) {
			logging.S.Debug(err)
			return false, err
		}
		logging.S.Warn(err.Error())
	}

	hostConfig.AccessKey = ""
	hostConfig.PolymorphicID = ""
	hostConfig.Name = ""

	_ = client.Logout()

	logging.S.Debugf("logged out of server [%s]", hostConfig.Host)
	return true, nil
}

func logout(clear bool, server string, all bool) error {
	config, err := readConfig()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return nil
		}

		return err
	}

	switch {
	case all:
		logging.S.Debug("logging out of all servers\n")
	case server == "":
		logging.S.Debug("logging out of current server\n")
	default:
		logging.S.Debugf("logging out of server [%s]\n", server)
	}

	if all {
		var logoutErr error
		for i := range config.Hosts {
			if _, err = logoutOfServer(&config.Hosts[i]); err != nil {
				logoutErr = err
			}
		}
		if logoutErr != nil {
			return errors.New("Failed to logout of all servers due to an internal error. Run with '--log-level=debug' for more info.")
		}

		fmt.Fprintf(os.Stderr, "Logged out of all servers.\n")
		if clear {
			config.Hosts = nil
			logging.S.Debug("cleared all servers from login list\n")
		}
	} else {
		for i := range config.Hosts {
			if (server == "" && config.Hosts[i].Current) || (server == config.Hosts[i].Host) {
				success, err := logoutOfServer(&config.Hosts[i])
				if err != nil {
					return fmt.Errorf("Failed to logout of server %s due to an internal error. Run with '--log-level=debug' for more info.", config.Hosts[i].Host)
				}
				if success {
					logging.S.Debugf("Logged out of server %s", config.Hosts[i].Host)
				}

				if clear {
					serverURL := config.Hosts[i].Host
					if len(config.Hosts) < 2 {
						config.Hosts = nil
					} else {
						config.Hosts[i] = config.Hosts[len(config.Hosts)-1]
						config.Hosts = config.Hosts[:len(config.Hosts)-1]
					}
					logging.S.Debugf("cleared server [%s]\n", serverURL)
				}
				break
			}
		}
	}

	if err := clearKubeconfig(); err != nil {
		return err
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	return nil
}
