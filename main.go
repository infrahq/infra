package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage: "Manage infrastructure identity & access",
		Commands: []*cli.Command{
			{
				Name:  "user",
				Usage: "Manage users",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add a new user",
						Action: func(c *cli.Context) error {
							fmt.Println("NOT IMPLEMENTED")
							return nil
						},
					},
					{
						Name:  "remove",
						Usage: "remove a user",
						Action: func(c *cli.Context) error {
							fmt.Println("NOT IMPLEMENTED")
							return nil
						},
					},
				},
			},
			{
				Name:  "login",
				Usage: "Login to an Infra Engine",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED")
					return nil
				},
			},
			{
				Name:  "logout",
				Usage: "Log out of an Infra Engine",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED")
					return nil
				},
			},
			{
				Name:  "start",
				Usage: "Start the Infra Engine",
				Action: func(c *cli.Context) error {
					cmd := exec.Command("envoy")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					if err != nil {
						log.Fatalf("cmd.Run() failed with %s\n", err)
					}
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
