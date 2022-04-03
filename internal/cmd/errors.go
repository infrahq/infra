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
	ErrUserNotFound      = errors.New(`no users found with this name`)
)

// user facing terminal constant errors - not meant for a stack trace, but a conversation
var (
	ErrTLSNotVerified = errors.New(`The authenticity of the host can't be established.`)
)

type FailedLoginError struct {
	LoggedInIdentity string
	LoginMethod      loginMethod
}

func (e *FailedLoginError) Error() string {
	var errorReason string

	switch e.LoginMethod {
	case localLogin:
		errorReason = "your id or password is incorrect"
	case accessKeyLogin:
		errorReason = "your access key is not valid"
	case OIDCLogin:
		errorReason = "could not login to infra through this provider"
	}

	msg := fmt.Sprintf("Login failed: %s.", errorReason)
	if (isLoggedInCurrent() && e.LoggedInIdentity == "") || (!isLoggedInCurrent() && e.LoggedInIdentity != "") {
		panic("LoggedInIdentity cannot be set unless user is logged in")
	}
	if isLoggedInCurrent() {
		msg += fmt.Sprintf(" Your existing session as %s is still active.", e.LoggedInIdentity)
	}

	return msg
}
