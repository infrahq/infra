package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/infrahq/infra/internal/api"
)

type UseOptions struct {
	Name      string
	Namespace string
	Labels    []string `mapstructure:"labels"`
}

func use(options *UseOptions) error {
	config, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := defaultAPIClient()
	if err != nil {
		return err
	}

	users, err := client.ListUsers(config.Name)
	if err != nil {
		if errors.Is(err, api.ErrForbidden) {
			fmt.Fprintln(os.Stderr, "Session has expired.")

			if err = login(&LoginOptions{Current: true}); err != nil {
				return err
			}

			return use(options)
		}

		return err
	}

	// This shouldn't be possible but check nonetheless
	switch {
	case len(users) < 1:
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("User %q not found", config.Name)
	case len(users) > 1:
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("Found multiple users for %q, please contact your administrator", config.Name)
	}

	user := users[0]

	// first make sure kubeconfig is up to date
	if err := updateKubeconfig(user); err != nil {
		return err
	}

	return nil
}
