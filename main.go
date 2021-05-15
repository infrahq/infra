package main

import (
	"log"

	"github.com/infrahq/infra/internal/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
