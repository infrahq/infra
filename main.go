package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/infrahq/infra/internal/cmd"
	"github.com/infrahq/infra/internal/logging"
)

func main() {
	if err := cmd.Run(context.Background(), os.Args[1:]...); err != nil {
		var userErr cmd.Error
		var unknownAuthErr x509.UnknownAuthorityError

		switch {
		case errors.Is(err, terminal.InterruptErr):
			logging.Debugf("user interrupted the process")
		case errors.As(err, &userErr):
			fmt.Fprintln(os.Stderr, userErr.Error())
		case errors.As(err, &unknownAuthErr):
			// Cert error is most likely caused by mismatch between the client session and its cached server
			// eg) server was re-installed, but the client cache was not cleared
			fmt.Fprintf(os.Stderr, "Error: %v\n\nTo see if the issue is with the client, run `infra login` again OR `infra logout --clear` to clear session\n", err)
		default:
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		os.Exit(1)
	}
}
