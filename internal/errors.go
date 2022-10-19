package internal

import (
	"fmt"
)

// These should probably closely match the HTTP status codes (though not a requirement),
// as they're base types that will be used to determine the correct HTTP response code.
var (
	// ErrUnauthorized refers to the http response code unauthorized, which really means not authenticated, despite its name. See https://stackoverflow.com/a/6937030/155585
	ErrUnauthorized = fmt.Errorf("unauthorized")
	// ErrBadGateway means an invalid response was received from an upstream server (probably an OIDC provider)
	ErrBadGateway = fmt.Errorf("bad gateway")

	ErrInternalServerError = fmt.Errorf("internal server error")
	ErrNotFound            = fmt.Errorf("record not found")
	ErrBadRequest          = fmt.Errorf("bad request")
	ErrNotImplemented      = fmt.Errorf("not implemented")
	ErrExpired             = fmt.Errorf("expired")
)
