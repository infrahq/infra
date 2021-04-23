package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/infrahq/infra/internal/server"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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

func readToken() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra"))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func writeToken(token string) error {
	// Create the .infra directory if it does not exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "token"), []byte(token), 0644); err != nil {
		return err
	}

	return nil
}

func Run() {
	app := &cli.App{
		Usage: "Infra: Identity Engine",
		Commands: []*cli.Command{
			{
				Name:  "users",
				Usage: "Manage users",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add a new user",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "username",
								Aliases:  []string{"u"},
								Required: true,
							},
							&cli.StringFlag{
								Name:     "password",
								Aliases:  []string{"p"},
								Required: true,
							},
						},
						Action: func(c *cli.Context) error {
							form := url.Values{}
							form.Add("username", c.String("username"))
							form.Add("password", c.String("password"))

							_, err := http.PostForm("http://localhost:3001/v1/users", url.Values{
								"username": {c.String("username")},
								"password": {c.String("password")},
							})
							if err != nil {
								panic("http request failed")
							}

							fmt.Println("User added")

							return nil
						},
					},
					{
						Name:    "list",
						Usage:   "list users",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							res, err := http.Get("http://localhost:3001/v1/users")
							if err != nil {
								panic("http request failed")
							}

							type response struct {
								Data []struct {
									ID       string `json:"id"`
									Username string `json:"username"`
									Created  int64  `json:"created"`
									Updated  int64  `json:"updated"`
								}
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								panic(err)
							}

							rows := [][]string{}
							for _, user := range decoded.Data {
								createdAt := time.Unix(user.Created, 0)

								rows = append(rows, []string{user.Username, user.ID, units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"})
							}

							PrintTable([]string{"USERNAME", "ID", "CREATED"}, rows)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a user",
						Action: func(c *cli.Context) error {
							req, err := http.NewRequest(http.MethodDelete, "http://localhost:3001/v1/users/"+c.Args().Get(0), nil)
							if err != nil {
								panic(err)
							}

							client := &http.Client{}
							res, err := client.Do(req)
							if err != nil {
								panic(err)
							}
							defer res.Body.Close()

							type response struct {
								ID      string `json:"id"`
								Deleted bool   `json:"deleted"`
								Error   string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								panic(err)
							}

							if decoded.Deleted {
								fmt.Println("User deleted")
							} else if len(decoded.Error) > 0 {
								fmt.Println("Could not delete user: " + decoded.Error)
							} else {
								fmt.Println("Could not delete user")
							}

							return nil
						},
					},
				},
			},
			{
				Name:  "login",
				Usage: "Login to an Infra Engine",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "username",
						Aliases:  []string{"u"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "password",
						Aliases:  []string{"p"},
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					form := url.Values{}
					form.Add("username", c.String("username"))
					form.Add("password", c.String("password"))

					res, err := http.Post("http://localhost:3001/v1/tokens", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
					if err != nil {
						panic("http request failed")
					}

					type tokenResponse struct {
						Token string `json:"token"`
					}

					var tr tokenResponse
					if err = json.NewDecoder(res.Body).Decode(&tr); err != nil {
						panic(err)
					}

					writeToken(tr.Token)

					loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
					// if you want to change the loading rules (which files in which order), you can do so here
					configOverrides := &clientcmd.ConfigOverrides{}
					// if you want to change override values or bind them to flags, there are methods to help you
					kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
					_, err = kubeConfig.RawConfig()

					// Create kubeconfig
					config := clientcmdapi.NewConfig()
					config.Clusters["infra"] = &clientcmdapi.Cluster{
						Server: "http://localhost:3001/v1/proxy",
					}
					config.AuthInfos["infra"] = &clientcmdapi.AuthInfo{
						Token: tr.Token,
					}
					config.Contexts["infra"] = &clientcmdapi.Context{
						Cluster:  "infra",
						AuthInfo: "infra",
					}
					config.CurrentContext = "infra"
					err = clientcmd.WriteToFile(*config, "config.yaml")
					fmt.Println("Kubeconfig updated")
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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "admin-password",
						Usage: "Initial admin password",
					},
					&cli.StringFlag{
						Name:  "domain", // TODO: should this be a comma-separated list of domains? or a wildcard? or subdomain?
						Usage: "Domain to use for LetsEncrypt certificates",
					},
				},
				Action: func(c *cli.Context) error {
					server.Run(&server.Options{
						AdminPassword: c.String("admin-password"),
						Domain:        c.String("domain"),
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
