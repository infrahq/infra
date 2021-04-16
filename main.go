package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/go-units"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func PrintTable(header []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data)
	table.Render()
}

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
					{
						Name:    "list",
						Usage:   "list users",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							type user struct {
								ID       string `json:"id"`
								Username string `json:"username"`
								Created  int64  `json:"created"`
								Updated  int64  `json:"updated"`
							}

							res, err := http.Get("http://localhost:3001/v1/users")
							if err != nil {
								panic("http request failed")
							}

							var users []user
							if err = json.NewDecoder(res.Body).Decode(&users); err != nil {
								panic(err)
							}

							rows := [][]string{}
							for _, user := range users {
								createdAt := time.Unix(user.Created, 0)

								rows = append(rows, []string{user.Username, user.ID, units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"})
							}

							PrintTable([]string{"USERNAME", "ID", "CREATED"}, rows)

							return nil
						},
					},
				},
			},
			{
				Name:  "login",
				Usage: "Login to an Infra Engine",
				Action: func(c *cli.Context) error {
					// Open login window

					//
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
				Name:  "server",
				Usage: "Start the Infra Engine",
				Action: func(c *cli.Context) error {
					Server()
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
