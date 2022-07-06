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
		case strings.HasSuffix(err.Error(), ": EOF"):
			logging.Debugf("%v\n", err)
			fmt.Fprintln(os.Stderr, "Could not reach infra server, please wait a moment and try again.")
		case errors.Is(err, terminal.InterruptErr):
			logging.Debugf("user interrupted the process")
		case errors.As(err, &userErr):
			fmt.Fprintln(os.Stderr, userErr.Error())
		default:
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		os.Exit(1)
	}
}
