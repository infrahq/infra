package main

import (
	"log"
	"os"

	"github.com/infrahq/infra/internal/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd(&cmd.CLI{})
	err := GenMarkdownFile(rootCmd, os.Stdout)
	if err != nil {
		log.Println(err.Error())
		return
	}
}
