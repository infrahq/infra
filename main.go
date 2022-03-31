package main

import (
	"errors"
	"os"

	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/infrahq/infra/internal/cmd"
	"github.com/infrahq/infra/internal/logging"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		if !errors.Is(err, terminal.InterruptErr) {
			logging.S.Error(err.Error())
		}
		os.Exit(1)
	}
}
