package main

import (
	"os"

	"github.com/infrahq/infra/internal/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
