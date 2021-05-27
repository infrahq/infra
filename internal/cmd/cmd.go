package cmd

import (
	"context"
	"crypto/tls"
	"sort"

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

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/docker/go-units"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server"
	"github.com/muesli/termenv"
	"github.com/olekukonko/tablewriter"
	"github.com/square/go-jose/jwt"
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

type Config struct {
	Host  string `json:"host"`
	Token string `json:"token"`
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

func unixHttpClient(path string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", path)
			},
		},
	}
}

func normalizeHost(host string) (string, error) {
	if host == "" {
		return "http://unix", nil
	}

	host = strings.Replace(host, "http://", "", -1)
	host = strings.Replace(host, "https://", "", -1)

	u, err := url.Parse("https://" + host)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

type TokenTransport struct {
	Token     string
	Transport http.RoundTripper
}

func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.Token)
	}
	return t.Transport.RoundTrip(req)
}

func blue(s string) string {
	return termenv.String(s).Bold().Foreground(termenv.ColorProfile().Color("#0057FF")).String()
}

func client(host string, token string, insecure bool) (client *http.Client, err error) {
	if host == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		return unixHttpClient(filepath.Join(homeDir, ".infra", "infra.sock")), nil
	} else if insecure {
		return &http.Client{
			Transport: &TokenTransport{
				Token: token,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
		}, nil
	} else {
		return &http.Client{
			Transport: &TokenTransport{
				Token:     token,
				Transport: http.DefaultTransport,
			},
		}, nil
	}
}

func Run() error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	token := config.Token

	host, err := normalizeHost(config.Host)
	if err != nil {
		host = config.Host
	}

	app := &cli.App{
		Usage: "manage user & machine access to Kubernetes",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "insecure",
				Aliases: []string{"i"},
				Usage:   "ignore tls warnings",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "login",
				Usage:     "Log in to Infra",
				ArgsUsage: "HOST",
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						cli.ShowCommandHelp(c, "create")
						return nil
					}

					host, err := normalizeHost(c.Args().First())
					if err != nil {
						return err
					}

					parsed, err := url.Parse(host)
					if err != nil {
						return err
					}

					hostname := parsed.Hostname()

					httpClient, err := client(host, token, c.Bool("insecure"))
					if err != nil {
						return err
					}

					res, err := httpClient.Get(host + "/v1/providers")
					if err != nil {
						return err
					}

					var response server.RetrieveProvidersResponse
					if err = checkAndDecode(res, &response); err != nil {
						return err
					}

					// TODO (jmorganca): clean this up to check a list based on the api results
					okta := fmt.Sprintf("Okta [%s]", response.Okta.Domain)
					userpass := "Username & password"

					options := []string{}
					if response.Okta.ClientID != "" && response.Okta.Domain != "" {
						options = append(options, okta)
					}
					options = append(options, userpass)

					var option string

					if len(options) > 1 {
						prompt := &survey.Select{
							Message: "Choose a login provider",
							Options: options,
						}
						err = survey.AskOne(prompt, &option, survey.WithIcons(func(icons *survey.IconSet) {
							icons.Question.Text = blue("?")
						}))
						if err == terminal.InterruptErr {
							return nil
						}
					} else {
						option = options[0]
					}

					form := url.Values{}

					switch {
					// Okta
					case option == okta:
						// Start OIDC flow
						// Get auth code from Okta
						// Send auth code to Infra to log in as a user
						state := generate.RandString(12)
						authorizeUrl := "https://" + response.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + response.Okta.ClientID + "&response_type=code&scope=openid+email&nonce=" + generate.RandString(10) + "&state=" + state

						fmt.Println(blue("✓") + " Logging in with Okta...")
						ls, err := newLocalServer()
						if err != nil {
							return err
						}

						err = browser.OpenURL(authorizeUrl)
						if err != nil {
							return err
						}

						code, recvstate, err := ls.wait()
						if err != nil {
							return err
						}

						if state != recvstate {
							return errors.New("received state is not the same as sent state")
						}

						form.Add("okta-code", code)

					case option == userpass:
						email := ""
						emailPrompt := &survey.Input{
							Message: "Email",
						}
						err = survey.AskOne(emailPrompt, &email, survey.WithShowCursor(true), survey.WithValidator(survey.Required), survey.WithIcons(func(icons *survey.IconSet) {
							icons.Question.Text = blue("?")
						}))
						if err == terminal.InterruptErr {
							return nil
						}

						password := ""
						passwordPrompt := &survey.Password{
							Message: "Password",
						}
						err = survey.AskOne(passwordPrompt, &password, survey.WithShowCursor(true), survey.WithValidator(survey.Required), survey.WithIcons(func(icons *survey.IconSet) {
							icons.Question.Text = blue("?")
						}))
						if err == terminal.InterruptErr {
							return nil
						}

						form.Add("email", email)
						form.Add("password", password)

						fmt.Println(blue("✓") + " Logging in with username & password...")
					}

					req, err := http.NewRequest("POST", host+"/v1/tokens", strings.NewReader(form.Encode()))
					if err != nil {
						return err
					}

					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

					res, err = httpClient.Do(req)
					if err != nil {
						return err
					}

					var createTokenResponse server.CreateTokenResponse
					err = checkAndDecode(res, &createTokenResponse)
					if err != nil {
						return err
					}

					fmt.Println(blue("✓") + " Logged in...")

					if err = writeConfig(&Config{
						Host:  hostname,
						Token: createTokenResponse.Token,
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

					config.Clusters[hostname] = &clientcmdapi.Cluster{
						Server: host + "/v1/proxy",
					}

					if c.Bool("insecure") {
						config.Clusters[hostname].InsecureSkipTLSVerify = true
					}

					config.AuthInfos[hostname] = &clientcmdapi.AuthInfo{
						Token: createTokenResponse.Token,
					}
					config.Contexts[hostname] = &clientcmdapi.Context{
						Cluster:  hostname,
						AuthInfo: hostname,
					}
					config.CurrentContext = hostname

					if err = clientcmd.WriteToFile(config, clientcmd.RecommendedHomeFile); err != nil {
						return err
					}

					fmt.Println(blue("✓") + " Kubeconfig updated")

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
						ArgsUsage: "EMAIL PASSWORD",
						Action: func(c *cli.Context) error {
							if c.NArg() <= 1 {
								cli.ShowCommandHelp(c, "create")
								return nil
							}

							email := c.Args().Get(0)
							password := c.Args().Get(1)
							form := url.Values{}
							form.Add("email", email)
							form.Add("password", password)

							httpClient, err := client(host, token, c.Bool("insecure"))
							if err != nil {
								return err
							}

							res, err := httpClient.PostForm(host+"/v1/users", form)
							if err != nil {
								return err
							}

							var response server.CreateUserResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							return nil
						},
					},
					{
						Name:    "list",
						Usage:   "list users",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							httpClient, err := client(host, token, c.Bool("insecure"))
							if err != nil {
								return err
							}

							res, err := httpClient.Get(host + "/v1/users")
							if err != nil {
								return err
							}

							var response server.ListUsersResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							email := ""
							if token != "" {
								tok, err := jwt.ParseSigned(token)
								if err != nil {
									return err
								}
								out := make(map[string]interface{})
								if err := tok.UnsafeClaimsWithoutVerification(&out); err != nil {
									return err
								}
								email = out["email"].(string)
							}

							sort.Slice(response.Data, func(i, j int) bool {
								return response.Data[i].Created > response.Data[j].Created
							})

							rows := [][]string{}
							for _, user := range response.Data {
								star := ""
								if user.Email == email {
									star = "*"
								}
								rows = append(rows, []string{user.Email + star, user.Provider, user.Permission, units.HumanDuration(time.Now().UTC().Sub(time.Unix(user.Created, 0))) + " ago"})
							}

							printTable([]string{"EMAIL", "PROVIDER", "PERMISSION", "CREATED"}, rows)

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
							req, err := http.NewRequest(http.MethodDelete, host+"/v1/users/"+user, nil)
							if err != nil {
								log.Fatal(err)
							}

							httpClient, err := client(host, token, c.Bool("insecure"))
							if err != nil {
								return err
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
							return nil
						},
					},
				},
			},
			{
				Name:  "server",
				Usage: "Start Infra server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "db",
						Usage: "Directory to store database",
						Value: filepath.Join(homeDir, ".infra", "db"),
					},
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Path to configuration file",
					},
					&cli.StringFlag{
						Name:  "tls-cache",
						Usage: "Directory to store cached letsencrypt & self-signed TLS certificates",
						Value: filepath.Join(homeDir, ".infra", "cache"),
					},
					&cli.BoolFlag{
						Name:  "ui",
						Usage: "Enable ui",
					},
					&cli.BoolFlag{
						Name:  "ui-dev",
						Usage: "Proxy to a development ui",
					},
				},
				Action: func(c *cli.Context) error {
					return server.Run(&server.ServerOptions{
						DBPath:     c.String("db"),
						ConfigPath: c.String("config"),
						TLSCache:   c.String("tls-cache"),
						UI:         c.Bool("ui"),
						UIDev:      c.Bool("ui-dev"),
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
