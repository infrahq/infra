package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/infrahq/infra/internal/cmd"
	"github.com/infrahq/infra/internal/logging"
)

func main() {
	if err := cmd.Run(context.Background(), os.Args[1:]...); err != nil {
		var userErr cmd.Error
		switch {
		case errors.Is(err, terminal.InterruptErr):
			logging.Debugf("user interrupted the process")
		case errors.As(err, &userErr):
			fmt.Fprintln(os.Stderr, userErr.Error())
		case strings.Contains(err.Error(), "x509: certificate signed by unknown authority"):
			// Cert error is most likely caused by mismatch between the client session and its cached server
			// eg) server was re-installed, but the client cache was not cleared
			fmt.Fprintf(os.Stderr, "Session certificate is no longer valid for this server; run 'infra login' to start a new session:\n\n%v\n", err)
		default:
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		os.Exit(1)
	}
}
