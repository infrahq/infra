package main

import (
	"context"
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
		switch {
		case errors.Is(err, terminal.InterruptErr):
			logging.S.Debug("user interrupted (kill, stop) the process")
		case errors.As(err, &userErr):
			fmt.Fprintln(os.Stderr, userErr.Error())
		default:
			fmt.Fprintf(os.Stderr, "Internal error:\n%v\n", err)
		}

		os.Exit(1)
	}
}
