package cmd

import (
	"context"
	"fmt"
	"io"
	logsyslog "log/syslog"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/linux"
)

func newSSHDCmd(cli *CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "sshd",
		Short:  "Commands for integrating with ssh server",
		Hidden: true,
	}

	cmd.AddCommand(newSSHDAuthKeysCmd(cli))
	return cmd
}

type sshAuthKeysOptions struct {
	fingerprint    string
	username       string
	configFilename string
}

func newSSHDAuthKeysCmd(cli *CLI) *cobra.Command {
	var opts sshAuthKeysOptions
	cmd := &cobra.Command{
		Use:    "auth-keys",
		Hidden: true,
		Args:   ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger(cli)

			// sshd_config: AuthorizedKeysCommand infra ssh auth-keys %u %f
			opts.username = args[0]
			opts.fingerprint = args[1]

			// log the error to syslog, since we expect stdout/stderr to be hidden
			if err := runSSHDAuthKeys(cli, logger, opts); err != nil {
				logger.Err(err).Msg("ssh auth-keys exit")
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.configFilename, "config-file",
		"/etc/infra/connector.yaml", "Path to connector config file")
	return cmd
}

func setupLogger(cli *CLI) zerolog.Logger {
	out := []io.Writer{cli.Stderr}
	// TODO: log to stderr if this fails?
	syslog, _ := logsyslog.New(logsyslog.LOG_AUTH|logsyslog.LOG_WARNING, "infra-ssh")
	if syslog != nil {
		out = append(out, zerolog.SyslogLevelWriter(syslog))
	}
	return zerolog.New(zerolog.MultiLevelWriter(out...))
}

func runSSHDAuthKeys(cli *CLI, logger zerolog.Logger, opts sshAuthKeysOptions) error {
	ctx := context.Background()
	logger.Debug().
		Str("username", opts.username).
		Str("fingerprint", opts.fingerprint).
		Msgf("Running infra ssh auth-keys")

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

	user, err := verifyUsernameAndFingerprint(ctx, logger, client, opts)
	if err != nil {
		return err
	}

	if err := verifyUserIsManagedByInfra(opts.username); err != nil {
		return err
	}

	if err := authorizeUserForDestination(ctx, client, user, config.Name); err != nil {
		return err
	}

	for _, key := range user.PublicKeys {
		// TODO: add expiration to this output when it's available
		cli.Output("%v %v", key.KeyType, key.PublicKey)
		logger.Debug().Msgf("%v %v", key.KeyType, key.PublicKey)
	}
	return nil
}

func verifyUsernameAndFingerprint(
	ctx context.Context,
	logger zerolog.Logger,
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
	logger.Debug().Msgf("user=%v (%v) pub keys %v", user.Name, user.ID, len(user.PublicKeys))

	if user.SSHUsername != opts.username {
		return nil, fmt.Errorf("public key is for a different user")
	}
	return &user, nil
}

// etcPasswdFilename is a shim for testing.
var etcPasswdFilename = "/etc/passwd"

func verifyUserIsManagedByInfra(username string) error {
	localUsers, err := linux.ReadLocalUsers(etcPasswdFilename)
	if err != nil {
		return fmt.Errorf("read users: %w", err)
	}

	for _, lu := range localUsers {
		if lu.Username == username && lu.IsManagedByInfra() {
			return nil
		}
	}
	return fmt.Errorf("user is not managed by infra")
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
