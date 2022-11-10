package cmd

import (
	"context"
	"fmt"
	logsyslog "log/syslog"

	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/cmd/cliopts"
)

var syslog *logsyslog.Writer

func init() {
	// TODO: log to stderr if this fails?
	syslog, _ = logsyslog.New(logsyslog.LOG_AUTH|logsyslog.LOG_WARNING, "infra-ssh")
}

type sshAuthKeysOptions struct {
	fingerprint    string
	username       string
	configFilename string
}

func newSSHAuthKeysCmd(cli *CLI) *cobra.Command {
	var opts sshAuthKeysOptions
	cmd := &cobra.Command{
		Use:    "auth-keys",
		Hidden: true,
		Args:   ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// sshd_config: AuthorizedKeysCommand infra ssh auth-keys %u %f
			opts.username = args[0]
			opts.fingerprint = args[1]

			// log the error to syslog, since we expect stdout/stderr to be hidden
			if err := runSSHAuthKeys(cli, opts); err != nil {
				syslog.Err(err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.configFilename, "config-file",
		"/etc/infra/connector.yaml", "Path to connector config file")
	return cmd
}

func runSSHAuthKeys(cli *CLI, opts sshAuthKeysOptions) error {
	ctx := context.Background()
	syslog.Info("Running infra ssh auth-keys")
	syslog.Debug(fmt.Sprintf("Fingerprint=%v Username=%v", opts.fingerprint, opts.username))

	// This command only uses a small subset of these options, but its expected
	// that it runs in the same environment as the ssh connector, so may as well
	// share the config.
	config := defaultConnectorOptions()
	err := cliopts.Load(&config, cliopts.Options{
		Filename:  opts.configFilename,
		EnvPrefix: "INFRA_CONNECTOR",
	})
	if err != nil {
		return err
	}

	client := config.APIClient()
	client.Name = "ssh-auth-keys-cmd"

	user, err := verifyUsernameAndFingerprint(ctx, client, opts)
	if err != nil {
		return err
	}

	if err := authorizeUserForDestination(ctx, client, user, config.Name); err != nil {
		return err
	}

	for _, key := range user.PublicKeys {
		// TODO: add expiration to this output when it's available
		cli.Output("%v %v", key.KeyType, key.PublicKey)
		syslog.Debug(fmt.Sprintf("%v %v", key.KeyType, key.PublicKey))
	}
	return nil
}

func verifyUsernameAndFingerprint(
	ctx context.Context,
	client *api.Client,
	opts sshAuthKeysOptions,
) (*api.User, error) {
	users, err := client.ListUsers(ctx, api.ListUsersRequest{
		PublicKeyFingerprint: opts.fingerprint,
	})
	if err != nil {
		return nil, fmt.Errorf("api list users: %w", err)
	}
	if len(users.Items) != 1 {
		return nil, fmt.Errorf("wrong number of users found %d", len(users.Items))
	}

	user := users.Items[0]
	syslog.Debug(fmt.Sprintf("user=%v (%v) pub keys %v", user.Name, user.ID, len(user.PublicKeys)))

	if user.SSHUsername != opts.username {
		return nil, fmt.Errorf("public key is for a different user")
	}
	return &user, nil
}

func authorizeUserForDestination(
	ctx context.Context,
	client *api.Client,
	user *api.User,
	name string,
) error {
	grants, err := client.ListGrants(ctx, api.ListGrantsRequest{
		User:          user.ID,
		Destination:   name,
		ShowInherited: true,
	})
	if err != nil {
		return err
	}
	if len(grants.Items) == 0 {
		return fmt.Errorf("user %v (%v) has not been granted access to this destination",
			user.Name, user.ID)
	}
	return nil
}
