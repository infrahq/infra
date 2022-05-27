package cmd

import (
	"fmt"
	"strings"
)

// CLI Errors are user facing errors that are formatted.
// Should be used for communication, rather than a stacktrace.
type Error struct {
	// OriginalError is an error that is caused by the system, used for logging/debugging
	// Only set this if you need it to be printed as part of the user facing 'Message'.
	OriginalError error

	// Message is a human readable error message that user will read on the CLI.
	// These should be full sentences, rather than a stack trace.
	// Consider including suggestions to resolve the error.
	Message string
}

func (e Error) Error() string {
	if e.OriginalError != nil {
		if len(e.Message) == 0 {
			return fmt.Sprintf("Error: %v", e.OriginalError)
		}

		// Strip '.' at the end when message includes the original error
		return fmt.Sprintf("%s: %v", strings.TrimSuffix(e.Message, "."), e.OriginalError)
	}

	return e.Message
}

func (e Error) Unwrap() error {
	return e.OriginalError
}
