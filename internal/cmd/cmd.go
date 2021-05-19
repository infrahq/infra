package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server"

	"github.com/cli/browser"
	"github.com/docker/go-units"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func printTable(header []string, data [][]string) {
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

func unixHttpClient(path string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", path)
			},
		},
	}
}

func normalizeHost(host string) string {
	if host == "" {
		return "http://unix"
	}

	u, err := url.Parse(host)
	if err != nil {
		return host
	}

	u.Scheme = "https"

	return u.String()
}

type Config struct {
	Host    string `json:"host"`
	User    string `json:"user"`
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

func readConfig() (config *Config, err error) {
	config = &Config{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "config"))
	if os.IsNotExist(err) {
		return config, nil
	}

	if err != nil {
		return
	}

	if err = json.Unmarshal(contents, &config); err != nil {
		return
	}

	return
}

func writeConfig(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "config"), []byte(contents), 0644); err != nil {
		return err
	}

	return nil
}

type BasicAuthTransport struct {
	Username string
	Password string
}

func (bat BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", bat.Username, bat.Password)))))
	return http.DefaultTransport.RoundTrip(req)
}

func (bat *BasicAuthTransport) Client() *http.Client {
	return &http.Client{Transport: bat}
}

func checkAndDecode(res *http.Response, i interface{}) error {
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	er := &server.ErrorResponse{}
	err = json.Unmarshal(data, &er)
	if err != nil {
		return err
	}

	if er.Error != "" {
		return errors.New(er.Error)
	}

	if res.StatusCode >= http.StatusBadRequest {
		return errors.New("received error status code: " + http.StatusText(res.StatusCode))
	}

	return json.Unmarshal(data, &i)
}

func Run() error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	host := config.Host
	token := config.Token

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	httpClient := &http.Client{}
	if config.Host == "" {
		httpClient = unixHttpClient(filepath.Join(homeDir, ".infra", "infra.sock"))
	} else {
		bat := BasicAuthTransport{
			Username: token,
			Password: "",
		}
		httpClient = bat.Client()
	}

	app := &cli.App{
		Usage: "manage user & machine access to Kubernetes",
		Commands: []*cli.Command{
			{
				Name:      "login",
				Usage:     "Log in to an Infra Engine",
				ArgsUsage: "HOST",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "token",
						Aliases: []string{"t"},
						Usage:   "token to authenticate with",
					},
				},
				Action: func(c *cli.Context) error {
					hostArg := c.Args().First()
					client := &http.Client{}

					if hostArg == "" {
						hostArg = host
						client = httpClient
					}

					// Get token from
					token := c.String("token")

					form := url.Values{}

					if token == "" {
						res, err := client.Get(normalizeHost(hostArg) + "/v1/providers")
						if err != nil {
							return err
						}

						var response server.RetrieveProvidersResponse
						if err = checkAndDecode(res, &response); err != nil {
							return err
						}

						// Start OIDC flow
						// Get auth code from Okta
						// Send auth code to Infra to log in as a user
						state := generate.RandString(12)
						authorizeUrl := "https://" + response.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + response.Okta.ClientID + "&response_type=code&scope=openid+email&nonce=" + generate.RandString(10) + "&state=" + state

						fmt.Println("Opening browser window...")
						server, err := newLocalServer()
						if err != nil {
							return err
						}

						err = browser.OpenURL(authorizeUrl)
						if err != nil {
							return err
						}

						code, recvstate, err := server.wait()
						if err != nil {
							return err
						}

						if state != recvstate {
							return errors.New("received state is not the same as sent state")
						}

						form.Add("okta-code", code)
					}

					req, err := http.NewRequest("POST", normalizeHost(hostArg)+"/v1/tokens", strings.NewReader(form.Encode()))
					if err != nil {
						return err
					}

					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

					if token != "" {
						req.SetBasicAuth(token, "")
					}

					res, err := client.Do(req)
					if err != nil {
						return err
					}

					var response server.CreateTokenResponse
					err = checkAndDecode(res, &response)
					if err != nil {
						return err
					}

					if err = writeConfig(&Config{
						Host:    normalizeHost(hostArg),
						Token:   response.SecretToken,
						Expires: response.Token.Expires,
						User:    response.Token.User,
					}); err != nil {
						fmt.Println(err)
						return err
					}

					// Load default config and merge new config in
					loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
					defaultConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
					config, err := defaultConfig.RawConfig()
					if err != nil {
						return err
					}

					config.Clusters[hostArg] = &clientcmdapi.Cluster{
						Server: normalizeHost(hostArg) + "/v1/proxy",
					}
					config.AuthInfos[hostArg] = &clientcmdapi.AuthInfo{
						Token: response.SecretToken,
					}
					config.Contexts[hostArg] = &clientcmdapi.Context{
						Cluster:  hostArg,
						AuthInfo: hostArg,
					}
					config.CurrentContext = hostArg

					if err = clientcmd.WriteToFile(config, clientcmd.RecommendedHomeFile); err != nil {
						log.Fatal(err)
					}

					fmt.Println("Kubeconfig updated.")

					return nil
				},
			},
			{
				Name:  "users",
				Usage: "Manage users",
				Subcommands: []*cli.Command{
					{
						Name:      "create",
						Usage:     "Create a user",
						ArgsUsage: "EMAIL",
						Action: func(c *cli.Context) error {
							if c.NArg() <= 0 {
								cli.ShowCommandHelp(c, "create")
								return nil
							}

							email := c.Args().Get(0)
							form := url.Values{}
							form.Add("email", email)

							res, err := httpClient.PostForm(normalizeHost(host)+"/v1/users", form)
							if err != nil {
								return err
							}

							var response server.CreateUserResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							fmt.Println(response.ID)

							return nil
						},
					},
					{
						Name:    "list",
						Usage:   "list users",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							res, err := httpClient.Get(normalizeHost(host) + "/v1/users")
							if err != nil {
								return err
							}

							var response server.ListUsersResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							rows := [][]string{}
							for _, user := range response.Data {
								createdAt := time.Unix(user.Created, 0)

								providers := ""
								for i, p := range user.Providers {
									if i > 0 {
										providers += ","
									}
									providers += p
								}

								rows = append(rows, []string{user.ID, user.Email, units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago", providers, user.Permission})
							}

							printTable([]string{"USER", "EMAIL", "CREATED", "PROVIDERS", "PERMISSION"}, rows)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a user",
						Action: func(c *cli.Context) error {
							if c.NArg() <= 0 {
								cli.ShowCommandHelp(c, "delete")
								return nil
							}

							user := c.Args().Get(0)
							req, err := http.NewRequest(http.MethodDelete, normalizeHost(host)+"/v1/users/"+user, nil)
							if err != nil {
								log.Fatal(err)
							}

							res, err := httpClient.Do(req)
							if err != nil {
								log.Fatal(err)
							}
							defer res.Body.Close()

							var response server.DeleteResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							if response.Deleted {
								fmt.Println(response.ID)
							}

							return nil
						},
					},
				},
			},
			{
				Name:  "tokens",
				Usage: "Manage tokens",
				Subcommands: []*cli.Command{
					{
						Name:      "create",
						Usage:     "Create a token",
						ArgsUsage: "USER",
						Action: func(c *cli.Context) error {
							if c.NArg() <= 0 {
								cli.ShowCommandHelp(c, "create")
								return nil
							}

							user := c.Args().Get(0)

							form := url.Values{}
							form.Add("user", user)
							res, err := httpClient.PostForm(normalizeHost(host)+"/v1/tokens", form)
							if err != nil {
								return err
							}

							var response server.CreateTokenResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							fmt.Println(response.SecretToken)

							return nil
						},
					},
					{
						Name:    "list",
						Usage:   "list tokens",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							res, err := httpClient.Get(normalizeHost(host) + "/v1/tokens")
							if err != nil {
								return err
							}

							var response server.ListTokensResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							rows := [][]string{}
							for _, expanded := range response.Data {
								createdAt := time.Unix(expanded.Created, 0)
								expiresAt := time.Unix(expanded.Expires, 0)
								expires := ""
								if expiresAt.After(time.Now()) {
									expires = "In " + strings.ToLower(units.HumanDuration(time.Until(expiresAt)))
								} else {
									expires = units.HumanDuration(time.Since(expiresAt)) + " ago"
								}
								rows = append(rows, []string{expanded.ID, expanded.User.Email, units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago", expires})
							}

							printTable([]string{"TOKEN", "EMAIL", "CREATED", "EXPIRY"}, rows)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a token",
						Action: func(c *cli.Context) error {
							if c.NArg() <= 0 {
								cli.ShowCommandHelp(c, "delete")
								return nil
							}

							token := c.Args().Get(0)
							req, err := http.NewRequest(http.MethodDelete, normalizeHost(host)+"/v1/tokens/"+token, nil)
							if err != nil {
								log.Fatal(err)
							}

							res, err := httpClient.Do(req)
							if err != nil {
								log.Fatal(err)
							}
							defer res.Body.Close()

							var response server.DeleteResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							if response.Deleted {
								fmt.Println(response.ID)
							}

							return nil
						},
					},
				},
			},
			{
				Name:  "engine",
				Usage: "Start Infra Engine",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "db",
						Usage: "Path to database",
					},
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Path to configuration file",
					},
					&cli.StringFlag{
						Name:  "tls-cache",
						Usage: "TLS certficate cache",
					},
				},
				Action: func(c *cli.Context) error {
					return server.Run(&server.ServerOptions{
						DBPath:     c.String("db"),
						ConfigPath: c.String("config"),
						TLSCache:   c.String("tls-cache"),
					})
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil
}
