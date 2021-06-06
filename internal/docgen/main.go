package main

import (
	"log"
	"os"

	"github.com/infrahq/infra/internal/cmd"
)

func main() {
	f, err := os.Create("./docs/cli.md")
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer f.Close()

	rootCmd, err := cmd.NewRootCmd()
	if err != nil {
		log.Fatalf(err.Error())
	}

	GenMarkdownFile(rootCmd, f)
}
