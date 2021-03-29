package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage: "Manage infrastructure identity & access",
		Commands: []*cli.Command{
			{
				Name:  "users",
				Usage: "List users",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
					return nil
				},
			},
			{
				Name:  "groups",
				Usage: "List groups",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
					return nil
				},
			},
			{
				Name:  "roles",
				Usage: "List available roles",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
					return nil
				},
			},
			{
				Name:  "permissions",
				Usage: "List configured permissions",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
					return nil
				},
			},
			{
				Name:  "login",
				Usage: "Login to an Infra registry",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
					return nil
				},
			},
			{
				Name:  "logout",
				Usage: "Log out of an Infra registry",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
					return nil
				},
			},
			{
				Name:  "install",
				Usage: "Install Infra on a cluster",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
					return nil
				},
			},
			{
				Name:  "start",
				Usage: "Start the Infra Engine",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED", c.Args().First())
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
