package connector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/repeat"
)

func runSSHConnector(ctx context.Context, opts Options) error {
	client := opts.APIClient()

	// TODO: any reason to keep registering in the background?
	destination, err := registerSSHConnector(ctx, client, opts)
	if err != nil {
		return fmt.Errorf("failed to register destination: %w", err)
	}

	con := connector{
		client:      client,
		destination: destination,
		options:     opts,
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		backOff := &backoff.ExponentialBackOff{
			InitialInterval:     2 * time.Second,
			MaxInterval:         time.Minute,
			RandomizationFactor: 0.2,
			Multiplier:          1.5,
		}
		waiter := repeat.NewWaiter(backOff)
		fn := func(ctx context.Context, grants []api.Grant) error {
			return updateLocalUsers(ctx, client, grants)
		}
		return syncGrantsToDestination(ctx, con, waiter, fn)
	})

	return group.Wait()
}

func registerSSHConnector(ctx context.Context, client apiClient, opts Options) (*api.Destination, error) {
	hostKeys, err := readHostKeysFromDir("/etc/ssh")
	if err != nil {
		return nil, err
	}

	// TODO: support looking up IP address using net.InterfaceAddrs

	destination := &api.Destination{
		Name: opts.Name,
		Kind: "ssh",
		Connection: api.DestinationConnection{
			URL: opts.EndpointAddr.String(),
			CA:  api.PEM(hostKeys),
		},
		// TODO: Roles - the groups available on the system,
	}
	err = createOrUpdateDestination(ctx, client, destination)
	if err != nil {
		return nil, err
	}
	return destination, nil
}

// TODO: check /etc/ssh/sshd_config for HostKey, HostKeyAlgorithms
func readHostKeysFromDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dir: %w", err)
	}

	buf := new(strings.Builder)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_key.pub") {
			continue
		}

		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", err
		}

		if err := readHostKeys(raw, buf); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func readHostKeys(in []byte, out io.Writer) error {
	pub, _, _, _, err := ssh.ParseAuthorizedKey(in)
	if err != nil {
		return err
	}
	_, err = out.Write(ssh.MarshalAuthorizedKey(pub))
	return err
}

// TODO: grants for groups need to be resolved to a user somehow
func updateLocalUsers(ctx context.Context, client *api.Client, grants []api.Grant) error {
	byUserID := grantsByUserID(grants)

	// List all the local users
	localUsers, err := readLocalUsers("/etc/passwd")
	if err != nil {
		return err
	}

	// Compare that list to the grants to get a list to remove and a list to add
	toDelete := []localUser{}
	for _, user := range localUsers {
		if !user.IsManagedByInfra() {
			continue
		}
		infraUID := user.Info[0]
		if _, ok := byUserID[infraUID]; !ok {
			toDelete = append(toDelete, user)
			continue
		}
		delete(byUserID, infraUID)
	}

	// Remove users
	// TODO:

	// Add users
	for _, grant := range byUserID {
		user, err := client.GetUser(ctx, grant.User)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}

		if err := addLinuxUser(user); err != nil {
			return err
		}
	}

	return nil
}

func grantsByUserID(grants []api.Grant) map[string]api.Grant {
	result := make(map[string]api.Grant, len(grants))
	for _, grant := range grants {
		result[grant.ID.String()] = grant
	}
	return result
}

func addLinuxUser(user *api.User) error {
	args := []string{
		"--comment", fmt.Sprintf("%v,%v", user.ID, sentinelManagedByInfra),
		"-m", user.SSHUsername,
	}
	cmd := exec.Command("useradd", args...)
	// TODO: capture error to syslog
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type localUser struct {
	Username string
	UID      string
	GID      string
	Info     []string
	HomeDir  string
}

const sentinelManagedByInfra = "managed by infra"

func (u localUser) IsManagedByInfra() bool {
	return len(u.Info) > 1 && u.Info[1] == sentinelManagedByInfra
}

func readLocalUsers(filename string) ([]localUser, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close() // read-only file, safe to ignore errors
	scan := bufio.NewScanner(fh)

	var result []localUser
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			return nil, fmt.Errorf("invalid line contains less than 7 fields")
		}
		result = append(result, localUser{
			Username: fields[0],
			// field 1 is not used
			UID:     fields[2],
			GID:     fields[3],
			Info:    strings.FieldsFunc(fields[4], isRuneComma),
			HomeDir: fields[5],
			// field 6 is login shell
		})
	}
	return result, scan.Err()
}

func isRuneComma(r rune) bool {
	return r == ','
}
