package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/muesli/termenv"

	"github.com/infrahq/infra/internal/logging"
)

// internal errors
var (
	//lint:ignore ST1005, user facing error
	ErrConfigNotFound    = errors.New(`Could not read local credentials. Are you logged in? Use "infra login" to login`)
	ErrProviderNotUnique = errors.New(`more than one provider exists with this name`)
	ErrUserNotFound      = errors.New(`no user found with this name`)
)

// User facing constant errors: to let user know why their command failed. Not meant for a stack trace, but a readable output of the reason for failure.
var (
	ErrTLSNotVerified = errors.New(`The authenticity of the host can't be established.`)
)

type LoginError struct {
	Message string
}

func (e *LoginError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Login error: %s.", e.Message)

	hostConfig, err := currentHostConfig()
	if err != nil {
		logging.S.Debugf("current host config: %v", err)
		return sb.String()
	}

	if hostConfig.isLoggedIn() {
		fmt.Fprintf(&sb, " Your session as %s to %s is still active.", termenv.String(hostConfig.Name).Bold().String(), termenv.String(hostConfig.Host).Bold().String())
	}

	return sb.String()
}
