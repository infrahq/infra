package connector

import (
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

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/linux"
	"github.com/infrahq/infra/internal/logging"
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

// etcPasswdFilename is a shim for testing.
var etcPasswdFilename = "/etc/passwd"

// TODO: grants for groups need to be resolved to a user somehow
func updateLocalUsers(ctx context.Context, client apiClient, grants []api.Grant) error {
	byUserID := grantsByUserID(grants)

	localUsers, err := linux.ReadLocalUsers(etcPasswdFilename)
	if err != nil {
		return err
	}

	// Compare that list to the grants to get a list to remove and a list to add
	var toDelete []linux.LocalUser
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

	for _, user := range toDelete {
		if err := linux.RemoveUser(user); err != nil {
			return fmt.Errorf("remove user: %w", err)
		}
		logging.L.Info().Str("username", user.Username).Msg("removed user")
	}

	for _, grant := range byUserID {
		user, err := client.GetUser(ctx, grant.User)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}

		if err := linux.AddUser(user); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		logging.L.Info().Str("username", user.SSHUsername).Msg("created user")
	}

	return nil
}

func grantsByUserID(grants []api.Grant) map[string]api.Grant {
	result := make(map[string]api.Grant, len(grants))
	for _, grant := range grants {
		result[grant.User.String()] = grant
	}
	return result
}
