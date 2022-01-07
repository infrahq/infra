package internal

import (
	"fmt"
)

// These should probably closely match the HTTP status codes (though not a requirement),
// as they're base types that will be used to determine the correct HTTP response code.
var (
	ErrUnauthorized = fmt.Errorf("unauthorized")
	ErrForbidden    = fmt.Errorf("forbidden")

	ErrDuplicate  = fmt.Errorf("duplicate record")
	ErrNotFound   = fmt.Errorf("record not found")
	ErrBadRequest = fmt.Errorf("bad request")
)
