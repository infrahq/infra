package cmd

import "fmt"

// CLI Errors are user facing errors that are formatted.
// Should be used for communication, rather than a stacktrace.
type Error struct {
	// Error that is caused by the system, used for debugging
	// Only set this if you need it to be printed as part of the message
	OriginalError error

	// Human readable error message that user will read on the CLI.
	// These should be full sentences, rather than a stack trace.
	// May include suggestions.
	Message string
}

func (e Error) Error() string {
	if e.OriginalError != nil {
		if len(e.Message) == 0 {
			return fmt.Sprintf("Internal error:\n%v", e.OriginalError)
		}

		// Strip '.' at the end when message includes the original error
		if string(e.Message[len(e.Message)-1]) == "." {
			e.Message = e.Message[:len(e.Message)-1]
		}
		return fmt.Sprintf("%v:\n       %v", e.Message, e.OriginalError)
	}

	return e.Message
}

func (e Error) Unwrap() error {
	return e.OriginalError
}
