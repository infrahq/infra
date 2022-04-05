package cmd

import "errors"

var (
	//lint:ignore ST1005, user facing error
	ErrConfigNotFound    = errors.New(`Could not read local credentials. Are you logged in? Use "infra login" to login`)
	ErrProviderNotUnique = errors.New(`more than one provider exists with this name`)
	ErrIdentityNotFound  = errors.New(`no identity found with this name`)
	ErrTLSNotVerified    = errors.New(`The authenticity of the host can't be established.`)
)

// Standard user facing messages
var (
	CmdOptionOverlapMsg = "%s is specified twice. Ignoring %s and proceeding with '%s'"
	NoProviderFoundMsg  = "No provider found with name %s"
	NoIdentityFoundMsg  = "No identity found with name %s"
)

// Standard panic messages
var (
	DuplicateEntryPanic = "more than one %s found with name '%s', which should not be possible"
)
