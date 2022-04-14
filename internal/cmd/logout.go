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
		Use:   "logout [URL]",
		Short: "Log out of Infra",
		Example: `# Log out of current server
$ infra logout
		
# Log out of a specific server
$ infra logout INFRA_URL
		
# Logout of all servers
$ infra logout --all 
		
# Log out of current server and clear from list 
$ infra logout --clear
		
# Log out of a specific server and clear from list
$ infra logout URL --clear 
		
# Logout and clear list of all servers 
$ infra logout --all --clear`,
		Args:  cobra.MaximumNArgs(1),
		Group: "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if all {
					fmt.Fprintf(os.Stderr, "  Server is already specified. Ignoring flag [--all] and logging out of server %s.\n", args[0])
					all = false
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

func logoutOfServer(hostConfig *ClientHostConfig) {
	client, err := apiClient(hostConfig.Host, hostConfig.AccessKey, hostConfig.SkipTLSVerify)
	if err != nil {
		logging.S.Warn(err.Error())
	}

	hostConfig.AccessKey = ""
	hostConfig.PolymorphicID = ""
	hostConfig.Name = ""

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

	switch {
	case all:
		stateChanged := false
		for i := range config.Hosts {
			if config.Hosts[i].isLoggedIn() {
				logoutOfServer(&config.Hosts[i])
				stateChanged = true
			}
		}
		if stateChanged {
			fmt.Fprintf(os.Stderr, "  Logged out of all servers.\n")
		} else {
			fmt.Fprintf(os.Stderr, "  Not logged in to any servers.\n")
		}

		if clear {
			if config.Hosts != nil {
				config.Hosts = nil
				fmt.Fprintf(os.Stderr, "  Cleared list of all servers\n")
			} else {
				fmt.Fprintf(os.Stderr, "  No servers to clear.\n")
			}
		}
	case url == "":
		serverFound := false
		for i := range config.Hosts {
			if config.Hosts[i].Current {
				serverFound = true
				if config.Hosts[i].isLoggedIn() {
					logoutOfServer(&config.Hosts[i])
					fmt.Fprintf(os.Stderr, "  Logged out of server %s.\n", config.Hosts[i].Host)
				} else {
					fmt.Fprintf(os.Stderr, "  Not logged in to server %s.\n", config.Hosts[i].Host)
				}
				break
			}
		}
		if serverFound {
			if clear {
				var newHostConfigs []ClientHostConfig

				for i := range config.Hosts {
					if config.Hosts[i].Current {
						fmt.Fprintf(os.Stderr, "  Cleared [%s] from list of servers\n", config.Hosts[i].Host)
						continue
					}

					newHostConfigs = append(newHostConfigs, config.Hosts[i])
				}
				config.Hosts = newHostConfigs
			}
		} else {
			fmt.Fprintf(os.Stderr, "  No current session to log out from.\n")
		}
	case url != "":
		serverFound := false
		for i := range config.Hosts {
			if url == config.Hosts[i].Host {
				serverFound = true
				if config.Hosts[i].isLoggedIn() {
					logoutOfServer(&config.Hosts[i])
					fmt.Fprintf(os.Stderr, "  Logged out of server %s.\n", config.Hosts[i].Host)
				} else {
					fmt.Fprintf(os.Stderr, "  Not logged in to server %s.\n", url)
				}
			}
		}

		if serverFound {
			if clear {
				var newHostConfigs []ClientHostConfig

				for i := range config.Hosts {
					if url == config.Hosts[i].Host {
						fmt.Fprintf(os.Stderr, "  Cleared %s from list of servers.\n", config.Hosts[i].Host)
						continue
					}

					newHostConfigs = append(newHostConfigs, config.Hosts[i])
				}
				config.Hosts = newHostConfigs
			}
		} else {
			fmt.Fprintf(os.Stderr, "  No server with url %s found.\n", url)
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
