package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/infrahq/infra/internal/cmd"
)

func main() {

	if err := cmd.Run(context.Background(), os.Args[1:]...); err != nil {
		var userErr cmd.Error
		switch {
		case errors.Is(err, terminal.InterruptErr):
			// print nothing, the user initiated the exit
		case errors.As(err, &userErr):
			fmt.Fprintln(os.Stderr, userErr.Error())
		default:
			userErr.OriginalError = err
			fmt.Fprintln(os.Stderr, userErr.Error())
		}

		os.Exit(1)
	}
}
