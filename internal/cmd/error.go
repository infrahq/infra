package cmd

// CLI Errors are user facing errors that are formatted.
// Should be used for communication, rather than a stacktrace.
type Error struct {
	// Short redacted version of OriginalError, required if OriginalError is set
	Cause string

	// OriginalError is the error that bubbled up, used for logging/debugging
	// Only set this if you need it to be printed as part of the user facing 'Message'.
	OriginalError error

	// Human readable message to resolve the error. These should be full sentences.
	Suggestion string
}

// Format is one of the three:
// a) Error: Cause
//    OriginalError
//
//    Suggestion
//
// b) Error: Cause
//    Suggestion
//
// c) Suggestion
func (e Error) Error() string {
	if e.OriginalError == nil && len(e.Cause) == 0 {
		return e.Suggestion
	}

	output := "Error: " + e.Cause
	if e.OriginalError != nil {
		output += "\n" + e.OriginalError.Error()
	}

	if len(e.Suggestion) > 0 {
		output += "\n\n" + e.Suggestion
	}

	return output
}

func (e Error) Unwrap() error {
	return e.OriginalError
}

var unauthorizedError = Error{
	Cause:      "missing permissions to run this command",
	Suggestion: "Please make sure you have the correct grants.",
}
