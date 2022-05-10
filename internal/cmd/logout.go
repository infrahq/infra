package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
)

type logoutCmdOptions struct {
	clear  bool
	server string
	all    bool
}

func newLogoutCmd(_ *CLI) *cobra.Command {
	var options logoutCmdOptions

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
		Args:  MaxArgs(1),
		Group: "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if options.all {
					return fmt.Errorf("Argument [SERVER] and flag [--all] cannot be both specified.")
				}
				options.server = args[0]
			}
			return logout(options.clear, options.server, options.all)
		},
	}

	cmd.Flags().BoolVar(&options.clear, "clear", false, "clear from list of servers")
	cmd.Flags().BoolVar(&options.all, "all", false, "logout of all servers")

	return cmd
}

func logoutOfServer(hostConfig *ClientHostConfig) (success bool) {
	if !hostConfig.isLoggedIn() {
		logging.S.Debugf("requested but not logged in to server [%s]", hostConfig.Host)
		return false
	}

	client, err := apiClient(hostConfig.Host, hostConfig.AccessKey, hostConfig.SkipTLSVerify)
	if err != nil {
		return false
	}

	hostConfig.AccessKey = ""
	hostConfig.PolymorphicID = ""
	hostConfig.Name = ""

	err = client.Logout()
	switch {
	case api.ErrorStatusCode(err) == http.StatusUnauthorized:
		return false
	case err != nil:
		return false
	}

	logging.S.Debugf("logged out of server [%s]", hostConfig.Host)
	return true
}

func logout(clear bool, server string, all bool) error {
	switch {
	case all:
		logging.S.Debug("logging out of all servers\n")
	case server == "":
		logging.S.Debug("logging out of current server\n")
	default:
		logging.S.Debugf("logging out of server [%s]\n", server)
	}

	if all {
		return logoutAll(clear)
	}

	return logoutOne(clear, server)
}

func logoutAll(clear bool) error {
	config, err := readConfig()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return nil
		}

		return err
	}

	for i := range config.Hosts {
		logoutOfServer(&config.Hosts[i])
	}

	fmt.Fprintf(os.Stderr, "Logged out of all servers.\n")
	if clear {
		config.Hosts = nil
		logging.S.Debug("cleared all servers from login list\n")
	}

	if err := clearKubeconfig(); err != nil {
		return err
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	return nil
}

func logoutOne(clear bool, server string) error {
	config, err := readConfig()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return nil
		}

		return err
	}

	host, idx := findClientConfigHost(config, server)

	if host == nil {
		return nil
	}

	success := logoutOfServer(host)
	if success {
		fmt.Fprintf(os.Stderr, "Logged out of server %s\n", host.Host)
	}

	if clear {
		serverURL := host.Host
		config.Hosts[idx] = config.Hosts[len(config.Hosts)-1]
		config.Hosts = config.Hosts[:len(config.Hosts)-1]
		logging.S.Debugf("cleared server [%s]", serverURL)
	}

	if err := clearKubeconfig(); err != nil {
		return err
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	return nil
}

func findClientConfigHost(config *ClientConfig, server string) (*ClientHostConfig, int) {
	for i := range config.Hosts {
		if (server == "" && config.Hosts[i].Current) || (server == config.Hosts[i].Host) {
			return &config.Hosts[i], i
		}
	}
	return nil, -1
}
