package main

import (
	"os"

	"github.com/infrahq/infra/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
