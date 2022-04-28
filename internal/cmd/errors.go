package cmd

import (
	"errors"
)

// internal errors
var (
	//lint:ignore ST1005, user facing error
	ErrConfigNotFound    = errors.New("config not found")
	ErrProviderNotUnique = errors.New("more than one provider exists with this name")
	ErrIdentityNotFound  = errors.New("no identity found with this name")
)

// Standard panic messages: it should not be possible for a user to arrive at this state - hence there is a bug in the code.
var (
	DuplicateEntryPanic = "more than one %s found with name '%s', which should not be possible"
)

// User facing messages: to let user know the state they are in
var (
	NoProviderFoundMsg = "No provider found with name %s"
	NoIdentityFoundMsg = "No identity found with name %s"
)
