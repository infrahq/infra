package cmd

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slices"

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

func newSSHHostsCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hosts HOSTNAME PORT",
		Short: "Check if the host is known to infra",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, port := args[0], args[1]
			if err := runSSHHosts(cli, host, port); err != nil {
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

func runSSHHosts(cli *CLI, hostname, port string) error {
	ctx := context.Background()

	opts, err := defaultClientOpts()
	if err != nil {
		return err
	}
	client, err := NewAPIClient(opts)
	if err != nil {
		return err
	}

	// TODO: check a local file cache to avoid querying the server in all cases
	dests, err := client.ListDestinations(ctx, api.ListDestinationsRequest{Kind: "ssh"})
	if err != nil {
		return err
	}

	// Exit if the hostname is not known to infra
	destination := destinationForName(dests.Items, hostname, port)
	if destination == nil {
		return fmt.Errorf("no destination matching that hostname")
	}

	if err := setupDestinationSSHConfig(ctx, cli, destination); err != nil {
		return err
	}
	return nil
}

func destinationForName(dests []api.Destination, hostname, port string) *api.Destination {
	for _, dest := range dests {
		destHost, destPort := splitHostPortSSH(dest.Connection.URL)
		if hostname == destHost && port == destPort {
			return &dest
		}

		// TODO: match destination name as well?
	}
	return nil
}

func splitHostPortSSH(hostname string) (host, port string) {
	var err error
	host, port, err = net.SplitHostPort(hostname)
	if err != nil {
		return hostname, "22" // default port for ssh client connections is 22
	}
	return host, port
}

func writeInfraKnownHosts(infraSSHDir string, dest *api.Destination) error {
	filename := filepath.Join(infraSSHDir, "known_hosts")
	fh, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	hostname := dest.Connection.URL
	if host, _, err := net.SplitHostPort(hostname); err == nil {
		hostname = host
	}

	for _, key := range strings.Split(string(dest.Connection.CA), "\n") {
		if key == "" {
			continue
		}

		line := fmt.Sprintf("%v %v\n", hostname, key)
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

func setupDestinationSSHConfig(ctx context.Context, cli *CLI, destination *api.Destination) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("user home directory: %w", err)
	}
	infraSSHDir := filepath.Join(homeDir, ".ssh/infra")
	if err := mkdirAll(infraSSHDir); err != nil {
		return err
	}

	hostCfg, err := currentHostConfig()
	if err != nil {
		return err
	}
	opts, err := apiClientFromHostConfig(hostCfg)
	if err != nil {
		return err
	}
	client, err := NewAPIClient(opts)
	if err != nil {
		return err
	}

	user, err := client.GetUserSelf(ctx)
	if err != nil {
		return err
	}

	keyFilename, err := provisionSSHKey(ctx, provisionSSHKeyOptions{
		cli:         cli,
		client:      client,
		hostConfig:  hostCfg,
		infraSSHDir: infraSSHDir,
		user:        user,
	})
	if err != nil {
		return fmt.Errorf("create ssh keypair: %w", err)
	}

	if err := writeInfraKnownHosts(infraSSHDir, destination); err != nil {
		return fmt.Errorf("write known hosts: %w", err)
	}

	if err := writeDestinationSSHConfig(infraSSHDir, destination, user, keyFilename); err != nil {
		return fmt.Errorf("write infra ssh config: %w", err)
	}
	return nil
}

// mkdirAll creates a directory and all its parents with mode 0o700, or returns
// an error. mkdirAll is a wrapper around os.MkdirAll to provide a better error
// message when the path already exists as a file.
func mkdirAll(path string) error {
	err := os.MkdirAll(path, 0o700)
	pathError := &fs.PathError{}
	if errors.As(err, &pathError) && errors.Is(pathError.Err, syscall.ENOTDIR) {
		return fmt.Errorf("failed to create directory %v, the path already exists as a regular file",
			pathError.Path)
	}
	return err
}

type provisionSSHKeyOptions struct {
	cli         *CLI
	client      *api.Client
	infraSSHDir string
	hostConfig  *ClientHostConfig
	user        *api.User
}

func provisionSSHKey(ctx context.Context, opts provisionSSHKeyOptions) (string, error) {
	keysCfg, err := readKeysConfig(opts.infraSSHDir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		keysCfg = &keysConfig{}
	case err != nil:
		return "", err
	}

	org, err := opts.client.GetOrganizationSelf(ctx)
	if err != nil {
		return "", err
	}

	keysDir := filepath.Join(opts.infraSSHDir, "keys")
	existingKeys := matchingPublicKeys(keysCfg, opts.hostConfig, org.ID)
	for i, existing := range existingKeys {
		filename := filepath.Join(keysDir, existing.PublicKeyID)
		if !fileExists(filename) || !fileExists(filename+".pub") {
			// key doesn't exist locally
			keysCfg.Keys = slices.Delete(keysCfg.Keys, i, i+1)
			continue
		}

		if userPublicKeyContains(opts.user.PublicKeys, existing.PublicKeyID) {
			// TODO: check expiration when expiry is added
			// key exists locally and in the API
			return filename, nil
		}

		// key doesn't exist in the API
		fmt.Fprintf(opts.cli.Stderr,
			"Removing %v because it was expired or deleted from Infra", filename)
		if err := os.Remove(filename); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("removing deleted key %w", err)
		}
		if err := os.Remove(filename + ".pub"); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("removing deleted key %w", err)
		}
		keysCfg.Keys = slices.Delete(keysCfg.Keys, i, i+1)
	}

	if err := mkdirAll(keysDir); err != nil {
		return "", err
	}
	fmt.Fprintf(opts.cli.Stderr, "Creating a new RSA 4096 bit key pair in %v\n", keysDir)

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", fmt.Errorf("generate key pair: %w", err)
	}

	sshPubKey, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return "", err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	hostname, _ := os.Hostname()
	resp, err := opts.client.AddUserPublicKey(ctx, &api.AddUserPublicKeyRequest{
		Name:      hostname,
		PublicKey: string(pubKeyBytes),
	})
	if err != nil {
		return "", fmt.Errorf("upload public key: %w", err)
	}

	keyFilename := filepath.Join(keysDir, resp.ID.String())
	fh, err := os.OpenFile(keyFilename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0o600)
	if err != nil {
		return "", err
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}
	if err := pem.Encode(fh, block); err != nil {
		return "", err
	}
	if err := fh.Close(); err != nil {
		return "", err
	}
	if err := os.WriteFile(keyFilename+".pub", pubKeyBytes, 0o600); err != nil {
		return "", err
	}

	keysCfg.Keys = append(keysCfg.Keys, localPublicKey{
		Server:         opts.hostConfig.Host,
		OrganizationID: org.ID.String(),
		UserID:         opts.hostConfig.UserID.String(),
		PublicKeyID:    resp.ID.String(),
	})
	if err := writeKeysConfig(opts.infraSSHDir, keysCfg); err != nil {
		return "", fmt.Errorf("write keys config: %w", err)
	}
	return keyFilename, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func updateUserSSHConfig(cli *CLI) error {
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
		_ = os.MkdirAll(userSSHDir, 0o700)
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

	if _, err := tmp.Write([]byte(infraUserSSHConfig)); err != nil {
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
Your SSH config at %v has been updated to connect to Infra SSH destinations.
This is a one time change. You do not need to use --enable-ssh again, and you
will not see this message again unless you remove your SSH config.

SSH destinations are in technical preview. If you have any problems or suggestions
we would love your feedback!

`,
		filename)
	return nil
}

const infraUserSSHConfig = `

Match exec "infra ssh hosts %h %p"
    Include ~/.ssh/infra/config

`

var infraDestinationSSHConfigTemplate = template.Must(template.New("ssh-config").
	Parse(infraDestinationSSHConfig))

const infraDestinationSSHConfig = `

# This file is managed by Infra. Do not edit!

Host {{ .Hostname }}
    IdentityFile {{ .KeyFilename }}
    IdentitiesOnly yes
    UserKnownHostsFile {{ .InfraSSHDir }}/known_hosts
    User {{ .Username }}
    Port {{ .Port }}

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

func writeDestinationSSHConfig(
	infraSSHDir string,
	destination *api.Destination,
	user *api.User,
	keyFilename string,
) error {
	filename := filepath.Join(infraSSHDir, "config")
	fh, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	host, port := splitHostPortSSH(destination.Connection.URL)
	data := map[string]any{
		"Username":    user.SSHLoginName,
		"Hostname":    host,
		"Port":        port,
		"KeyFilename": keyFilename,
		"InfraSSHDir": infraSSHDir,
	}
	if err := infraDestinationSSHConfigTemplate.Execute(fh, data); err != nil {
		return err
	}
	if err := fh.Sync(); err != nil {
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	return nil
}
