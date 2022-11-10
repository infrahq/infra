package connector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	"github.com/infrahq/infra/internal/repeat"

	"github.com/infrahq/infra/api"
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
		return syncGrantsToDestination(ctx, con, waiter, updateLocalUsers)
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

func updateLocalUsers(ctx context.Context, grants []api.Grant) error {
	// TODO: grants for groups need to be resolved to a user somehow

	// List all users managed by infra
	localUsers, err := readLocalUsers("/etc/passwd")
	if err != nil {
		return err
	}

	_ = localUsers

	// Compare that list to the grants to get a list to remove and a list to add

	// Remove users
	// Add users

	return nil
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
