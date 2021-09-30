package main

import (
	"log"
	"os"

	"github.com/infrahq/infra/internal/cmd"
)

func main() {
	f, err := os.Create("./docs/cli.md")
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer f.Close()

	rootCmd, err := cmd.NewRootCmd()
	if err != nil {
		log.Println(err.Error())
		return
	}

	err = GenMarkdownFile(rootCmd, f)
	if err != nil {
		log.Println(err.Error())
		return
	}
}
