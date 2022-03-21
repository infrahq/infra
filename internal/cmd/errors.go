package cmd

import "errors"

var (
	//lint:ignore ST1005, user facing error
	ErrConfigNotFound    = errors.New(`Could not read local credentials. Are you logged in? Use "infra login" to login`)
	ErrProviderNotUnique = errors.New(`more than one provider exists with this name`)
	ErrUserNotFound      = errors.New(`no users found with this name`)
)
