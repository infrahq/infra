package main

import (
	"fmt"
	"log"
	"os"

	"github.com/infrahq/infra/internal/server"
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
					server.Run(&server.Options{
						Port: 3001,
					})
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
