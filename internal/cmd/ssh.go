package cmd

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
)

func newSSHCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh",
		Short:  "Commands for integrating with ssh",
		Hidden: true,
	}

	cmd.AddCommand(newSSHHostsCmd(cli))

	return cmd
}

func newSSHHostsCmd(*CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hosts",
		Short: "Check if the host is known to infra",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runSSHHosts(args[0]); err != nil {
				// Prevent an error from being printed to stderr, because it
				// is printed every time a user runs ssh for a non-infra host.
				logging.L.Debug().Err(err).Msg("exit from infra ssh hosts")
				return exitError{code: 1}
			}
			return nil
		},
	}
	return cmd
}

// exitCode is used to exit with a non-zero code without printing any error
// to stderr.
type exitError struct {
	code int
}

func (e exitError) ExitCode() int {
	return e.code
}

func (e exitError) Error() string {
	return fmt.Sprintf("exit code %v", e.code)
}

func runSSHHosts(hostname string) error {
	ctx := context.Background()

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	// TODO: check a local file cache to avoid querying the server in all cases
	dests, err := client.ListDestinations(ctx, api.ListDestinationsRequest{Kind: "ssh"})
	if err != nil {
		return err
	}

	// Exit if the hostname is not known to infra
	destination := destinationForName(dests.Items, hostname)
	if destination == nil {
		return fmt.Errorf("no destination matching that hostname")
	}

	if err := setupSSHConfig(ctx, dests.Items); err != nil {
		return err
	}
	return nil
}

func destinationForName(dests []api.Destination, hostname string) *api.Destination {
	for _, dest := range dests {
		if dest.Connection.URL == hostname {
			return &dest
		}

		// Try the url without port
		host, _, err := net.SplitHostPort(dest.Connection.URL)
		if err == nil && hostname == host {
			return &dest
		}
	}
	return nil
}

// TODO: write this file path to the infra config as well?
// TODO: only write one dest at a time, and leave the rest of the file.
func writeInfraKnownHosts(infraSSHDir string, dests []api.Destination) error {
	filename := filepath.Join(infraSSHDir, "known_hosts")
	fh, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	for _, dest := range dests {
		line := fmt.Sprintf("%v %v", dest.Connection.URL, dest.Connection.CA)
		if _, err := fh.WriteString(line); err != nil {
			return err
		}
	}
	if err := fh.Sync(); err != nil {
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	return nil
}

func setupSSHConfig(ctx context.Context, destinations []api.Destination) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("user home directory: %w", err)
	}
	infraSSHDir := filepath.Join(homeDir, ".ssh/infra")
	_ = os.MkdirAll(infraSSHDir, 0700)

	cfg, err := readConfig()
	if err != nil {
		return err
	}
	client, err := cfg.APIClient()
	if err != nil {
		return err
	}

	if err := provisionSSHKey(ctx, client, infraSSHDir); err != nil {
		return err
	}

	if err := writeInfraKnownHosts(infraSSHDir, destinations); err != nil {
		return fmt.Errorf("write known hosts: %w", err)
	}
	return nil
}

func provisionSSHKey(ctx context.Context, client *api.Client, infraSSHDir string) error {
	keyFilename := filepath.Join(infraSSHDir, "key")

	// TODO: check expiration
	// TODO: check the key exists in the API
	if fileExists(keyFilename) {
		return nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}

	fh, err := os.OpenFile(keyFilename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	block := &pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: priv}
	if err := pem.Encode(fh, block); err != nil {
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}

	sshPubKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		return err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	if err := os.WriteFile(keyFilename+".pub", pubKeyBytes, 0600); err != nil {
		return err
	}

	hostname, _ := os.Hostname()
	_, err = client.AddUserPublicKey(ctx, &api.AddUserPublicKeyRequest{
		Name:      hostname,
		PublicKey: string(pubKeyBytes),
	})
	return err
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func getSSHUsername(ctx context.Context) (string, error) {
	cfg, err := readConfig()
	if err != nil {
		return "", err
	}
	hostCfg, err := cfg.CurrentHostConfig()
	if err != nil {
		return "", err
	}
	client, err := cfg.APIClient()
	if err != nil {
		return "", err
	}
	user, err := client.GetUser(ctx, hostCfg.UserID)
	if err != nil {
		return "", err
	}
	return user.SSHUsername, nil
}

func updateUserSSHConfig(cli *CLI, sshUsername string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("user home directory: %w", err)
	}

	userSSHDir := filepath.Join(homeDir, ".ssh")
	filename := filepath.Join(userSSHDir, "config")

	var original []byte
	fh, err := os.Open(filename)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// file is missing, we'll create it later
		_ = os.MkdirAll(userSSHDir, 0700)
	case err != nil:
		return fmt.Errorf("open ssh config: %w", err)
	default:
		defer fh.Close() // closing the read only file

		if hasInfraMatchLine(fh) {
			return nil
		}

		_, err = fh.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("seek: %w", err)
		}

		original, err = io.ReadAll(fh)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		_ = fh.Close() // closing the read only file
	}

	tmp, err := os.CreateTemp(userSSHDir, "config-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(original); err != nil {
		return err
	}

	data := map[string]string{"Username": sshUsername}
	if err := infraSSHConfigTemplate.Execute(tmp, data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), filename); err != nil {
		return err
	}
	cli.Output(`
Your SSH config at %v has been created or updated to use 'infra ssh hosts' for
connecting to Infra SSH destinations.
`,
		filename)
	return nil
}

var infraSSHConfigTemplate = template.Must(template.New("ssh-config").Parse(infraSSHConfig))

const infraSSHConfig = `

Match exec "infra ssh hosts %h"
    IdentityFile ~/.ssh/infra/key
    IdentitiesOnly yes
    User {{ .Username }}
    UserKnownHostsFile ~/.ssh/infra/known_hosts

`

// hasInfraMatchLine does a minimal parse of the ssh client config file and
// returns true if it finds the "Match" line required to use infra ssh,
// otherwise returns false. Scanning errors are ignored.
//
// See https://man.openbsd.org/ssh_config.5 for details about the file format.
func hasInfraMatchLine(sshConfig io.Reader) bool {
	if sshConfig == nil {
		return false
	}

	// TODO: test many match lines that don't match
	lineScan := bufio.NewScanner(sshConfig)
	for lineScan.Scan() {
		fields := strings.Fields(lineScan.Text())
		if len(fields) < 3 {
			continue
		}
		if !strings.EqualFold(fields[0], "Match") {
			continue
		}
		if !strings.EqualFold(fields[1], "exec") {
			continue
		}
		cmd := strings.Join(fields[2:], " ")
		if strings.Contains(cmd, "infra ssh hosts") {
			return true
		}
	}
	if err := lineScan.Err(); err != nil {
		logging.Warnf("Failed to read ssh config: %v", err)
	}
	return false
}
