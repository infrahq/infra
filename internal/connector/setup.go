package connector

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/infrahq/infra/internal/logging"
)

func RunSetup(opts Options) error {
	switch opts.SetupKind {
	case "ssh":
		opts.Kind = "ssh"
	default:
		return fmt.Errorf("unexpected value for setup: %v", opts.SetupKind)
	}

	if err := validateOptionsSSH(opts); err != nil {
		// TODO: improve error message
		return err
	}

	if _, err := user.LookupGroup(opts.SSH.Group); err != nil {
		return fmt.Errorf("group %v does not exist, create it or set ssh.group", opts.SSH.Group)
	}

	if _, err := os.Stat(opts.SSH.SSHDConfigPath); err != nil {
		return fmt.Errorf("file %v does not exist or is not readable, set ssh.sshd_config_path to the sshd_config used by the ssh server, or run setup with sudo",
			opts.SSH.SSHDConfigPath)
	}

	config, err := readSSHDConfig(opts.SSH.SSHDConfigPath, "/etc/ssh")
	if err != nil {
		return err
	}

	keys, err := readSSHHostKeys(config.HostKeys, "/etc/ssh")
	if err != nil {
		return err
	}

	printSSHOptions(opts, keys)

	// TODO: prompt for confirmation? Or do we error earlier if there are problems?
	filename := "/etc/infra/connector.yaml"
	if err := writeConfig(opts, filename); err != nil {
		return err
	}

	if !slices.Contains(config.Includes, "/etc/ssh/sshd_config.d/*.conf") {
		logging.L.Warn().Msg(
			"sshd_config does not have expected wildcard include for /etc/ssh/sshd_config.d/*.conf. " +
				"You need to include /etc/infra/sshd_config.")
	} else {
		fmt.Println("copying infra sshd config to /etc/ssh/sshd_config.d/infra.conf")
		raw, err := os.ReadFile("/etc/infra/sshd_config")
		if err != nil {
			return err
		}

		err = os.WriteFile("/etc/ssh/sshd_config.d/infra.conf", raw, 0o600)
		if err != nil {
			return err
		}
	}

	fmt.Println("\nTesting sshd_config with sshd -t")
	// TODO: would be nice to use -T -C <user> if we had a user in the right group
	// TODO: should we inspect the printed config to ensure it's correct?
	if err := execCmd("sshd", "-t"); err != nil {
		return fmt.Errorf("sshd config test failed: %w", err)
	}

	fmt.Println("\nStarting and enabling the infra systemd service")
	if err := execCmd("systemctl", "start", "infra"); err != nil {
		return fmt.Errorf("failed to start systemd service: %w", err)
	}
	if err := execCmd("systemctl", "enable", "infra"); err != nil {
		return fmt.Errorf("failed to enable systemd service: %w", err)
	}
	return nil
}

// TODO: include host key filenames
// TODO: include endpointAddr
func printSSHOptions(opts Options, keys *hostKeys) {
	// TODO:
}

// TODO: how to specify target filename?
// TODO: how to omit irrelevant sections of the config?
func writeConfig(opts Options, filename string) error {
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}
	return yaml.NewEncoder(fh).Encode(opts)
}

func execCmd(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
