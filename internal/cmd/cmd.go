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

	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/internal/util"

	"github.com/cli/browser"
	"github.com/docker/go-units"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

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

func NewUnixHttpClient(path string) *http.Client {
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
		httpClient = NewUnixHttpClient(filepath.Join(homeDir, ".infra", "infra.sock"))
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

						type providersResponse struct {
							Okta struct {
								ClientID string `json:"client-id"`
								Domain   string `json:"domain"`
							}
							Error string `json:"error"`
						}

						var decoded providersResponse
						if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
							return err
						}

						// Start OIDC flow
						// Get auth code from Okta
						// Send auth code to Infra to log in as a user
						state := util.RandString(12)
						authorizeUrl := "https://" + decoded.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + decoded.Okta.ClientID + "&response_type=code&scope=openid+email&nonce=" + util.RandString(10) + "&state=" + state

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

						form.Add("code", code)
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

					body, err := ioutil.ReadAll(res.Body)
					if err != nil {
						return err
					}

					type tokenResponse struct {
						Token struct {
							ID      string `json:"id"`
							User    string `json:"user"`
							Created int64  `json:"created"`
							Updated int64  `json:"updated"`
							Expires int64  `json:"expires"`
						}
						SecretToken string `json:"secret_token"`
						Host        string `json:"host"`
						Error       string
					}

					var response tokenResponse
					if err = json.Unmarshal(body, &response); err != nil {
						return cli.Exit(err, 1)
					}

					if res.StatusCode != http.StatusCreated {
						return cli.Exit(response.Error, 1)
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
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "permission",
								Aliases: []string{"p"},
								Usage:   "Permission to assign user",
							},
						},
						Action: func(c *cli.Context) error {
							if c.NArg() <= 0 {
								cli.ShowCommandHelp(c, "create")
								return nil
							}

							email := c.Args().Get(0)

							form := url.Values{}
							form.Add("email", email)

							if c.String("permission") != "" {
								form.Add("permission", c.String("permission"))
							}

							res, err := httpClient.PostForm(normalizeHost(host)+"/v1/users", form)
							if err != nil {
								log.Fatal(err)
							}

							type response struct {
								ID      string `json:"id"`
								Email   string `json:"email"`
								Created int64  `json:"created"`
								Updated int64  `json:"updated"`
								Error   string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								panic(err)
							}

							form = url.Values{}
							form.Add("user", decoded.ID)
							res, err = httpClient.PostForm(normalizeHost(host)+"/v1/tokens", form)
							if err != nil {
								log.Fatal(err)
							}

							if res.StatusCode != http.StatusCreated {
								log.Fatal(decoded.Error)
							}

							type tokenResponse struct {
								data.Token
								SecretToken string `json:"secret_token"`
								Host        string `json:"host"`
								Error       string
							}

							var decodedTokenResponse tokenResponse
							if err = json.NewDecoder(res.Body).Decode(&decodedTokenResponse); err != nil {
								panic(err)
							}

							fmt.Println()
							fmt.Println("User " + decoded.Email + " added. Please share the following command with them so they can log in:")
							fmt.Println()
							if decodedTokenResponse.Host == "" {
								fmt.Println("infra login --token " + decodedTokenResponse.SecretToken)
							} else {
								fmt.Println("infra login --token " + decodedTokenResponse.SecretToken + " " + decodedTokenResponse.Host)
							}
							fmt.Println()

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

							type response struct {
								Data  []data.User
								Error string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								return err
							}

							if decoded.Error != "" {
								return err
							}

							rows := [][]string{}
							for _, user := range decoded.Data {
								createdAt := time.Unix(user.Created, 0)

								providers := ""
								for i, p := range user.Providers {
									if i > 0 {
										providers += ","
									}
									providers += p
								}

								rows = append(rows, []string{user.ID, providers, user.Email, units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago", user.Permission})
							}

							printTable([]string{"USER ID", "PROVIDERS", "EMAIL", "CREATED", "PERMISSION"}, rows)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a user",
						Action: func(c *cli.Context) error {
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

							type response struct {
								ID      string `json:"id"`
								Deleted bool   `json:"deleted"`
								Error   string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								return err
							}

							if decoded.Deleted {
								fmt.Println(decoded.ID)
							} else if len(decoded.Error) > 0 {
								return errors.New(decoded.Error)
							} else {
								return errors.New("could not delete user")
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
					return server.ServerRun(&server.ServerOptions{
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
