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
					fmt.Fprintf(os.Stderr, "Server is specified. Ignoring flag [--all] and logging out of server %s.", args[1])
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

	//To do : output change of state

	// Log out of server(s)
	stateChanged := false
	for i, hostConfig := range config.Hosts {
		if all || (url == hostConfig.Host) || (url == "" && hostConfig.Current) {
			logoutOfServer(hostConfig)

			// Clear user information
			config.Hosts[i].AccessKey = ""
			config.Hosts[i].PolymorphicID = ""
			config.Hosts[i].Name = ""

			if !all {
				fmt.Fprintf(os.Stderr, "  Logged out of server %s.\n", config.Hosts[i].Host)
			}
			stateChanged = true
		}
	}

	if !stateChanged {
		switch {
		case url == "":
			fmt.Fprintf(os.Stderr, "  No current session to log out from.\n")
		case all:
			fmt.Fprintf(os.Stderr, "  Not logged in to any server.\n")
		default:
			fmt.Fprintf(os.Stderr, "  Not logged in to server %s.\n", url)
		}
		return nil
	}

	if all {
		fmt.Fprintf(os.Stderr, "  Logged out of all servers.\n")
	}

	// Clear from list of saved servers
	var newHostConfigs []ClientHostConfig
	if clear {
		if all {
			config.Hosts = nil
			fmt.Fprintf(os.Stderr, "  Cleared list of all servers\n")
		} else {
			for i := range config.Hosts {
				if (url == config.Hosts[i].Host) || (url == "" && config.Hosts[i].Current) {
					fmt.Fprintf(os.Stderr, "  Cleared [%s] from list of servers\n", config.Hosts[i].Host)
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
