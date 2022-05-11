package cmd

import (
	"errors"
	"fmt"
)

// internal errors
var (
	//lint:ignore ST1005, user facing error
	ErrConfigNotFound    = errors.New(`Could not read local credentials. Are you logged in? Use "infra login" to login`)
	ErrProviderNotUnique = errors.New(`more than one provider exists with this name`)
	ErrUserNotFound      = errors.New(`no user found with this name`)
)

// Standard panic messages: it should not be possible for a user to arrive at this state - hence there is a bug in the code.
var (
	DuplicateEntryPanic = "more than one %s found with name '%s', which should not be possible"
)

// User facing messages: to let user know the state they are in
var (
	NoProviderFoundMsg = "No provider found with name %s"
	NoUserFoundMsg     = "No user found with name %s"
)

// User facing constant errors: to let user know why their command failed. Not meant for a stack trace, but a readable output of the reason for failure.
var (
	ErrTLSNotVerified = errors.New(`The authenticity of the host can't be established.`)
)

// User facing variable errors
type FailedLoginError struct {
	LoggedInIdentity string
	LoginMethod      loginMethod
}

func (e *FailedLoginError) Error() string {
	var errorReason string

	switch e.LoginMethod {
	case localLogin:
		errorReason = "your id or password may be incorrect"
	case accessKeyLogin:
		errorReason = "your access key is may not be valid"
	case oidcLogin:
		errorReason = "could not login to infra through this connected identity provider"
	}

	msg := fmt.Sprintf("Login failed: %s.", errorReason)
	if (isLoggedInCurrent() && e.LoggedInIdentity == "") || (!isLoggedInCurrent() && e.LoggedInIdentity != "") {
		panic("LoggedInIdentity cannot be set unless user is logged in")
	}
	if isLoggedInCurrent() {
		msg += fmt.Sprintf(" You are still logged in as [%s].", e.LoggedInIdentity)
	}

	return msg
}
