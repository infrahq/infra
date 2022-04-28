package cmd

import (
	"fmt"

	"github.com/infrahq/infra/internal/logging"
)

// UserFacingError wraps an error to provide better error messages to users.
// Any non-UserFacingError will be printed as "unhandled errors".
type UserFacingError struct {
	// original error
	Underlying error

	// formatted outputs that users see when running the CLI commands
	// should be a readable sentence rather than a stacktrace
	UserFacingMessage string

	// show or hide Underlying error from outputting with the UserFacingMessage
	ShowUnderlying bool
}

func (u UserFacingError) Error() string {
	// if apiError, ok := u.Underlying.(api.Error); ok {
	// 	return formatAPIError(apiError, u.UserFacingMessage)
	// }

	if u.ShowUnderlying {
		// Strip '.' at the end when printing underlying error
		if string(u.UserFacingMessage[len(u.UserFacingMessage)-1]) == "." {
			u.UserFacingMessage = u.UserFacingMessage[:len(u.UserFacingMessage)-1]
		}
		return fmt.Sprintf("%v:\n       %v", u.UserFacingMessage, u.Underlying)
	}

	if u.UserFacingMessage == "" {
		return fmt.Sprintf("Internal error:\n%v", u.Underlying.Error())
	}

	if u.Underlying != nil {
		logging.S.Debug(u.Underlying.Error())
	}
	return u.UserFacingMessage
}

func (u UserFacingError) Unwrap() error {
	return u.Underlying
}

// func formatAPIError(err error, message string) string {
// 	//ErrUnauthorized
// 	switch {
// 	case errors.Is(err, api.ErrUnauthorized):
// 		return fmt.Sprintf("%v: %v", message, err.Error())
// 	}
// 	switch apiError.Code {
// 	case http.StatusBadRequest:
// 		return fmt.Sprintf("%v: bad request: %v", message, apiError.Message)
// 	case http.StatusBadGateway:
// 		// this error should be displayed to the user so they can see its an external problem
// 		return fmt.Sprintf("%v: bad gateway: %v", message, apiError.Message)
// 	case http.StatusInternalServerError:
// 		return fmt.Sprintf("%v: internal error: %v", message, apiError.Message)
// 	case http.StatusGone:
// 		return fmt.Sprintf("%v: endpoint no longer exists, upgrade the CLI: %v", message, apiError.Message)
// 	default:
// 		return fmt.Sprintf("%v: request failed: %v", message, apiError.Message)
// 	}
// }

var (
	ErrTLSNotVerified UserFacingError = UserFacingError{
		UserFacingMessage: "The authenticity of the host can't be established.",
	}
)
