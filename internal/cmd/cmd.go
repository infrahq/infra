package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"

	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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
	"github.com/cli/browser"
	"github.com/docker/go-units"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server"
	"github.com/muesli/termenv"
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

type Config struct {
	Host  string `json:"host"`
	CA    string `json:"ca"`
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

	u.Path = "/api/v1/namespaces/infra/services/infra/proxy"

	return u.String(), nil
}

func GetCertificatesPEM(address string) (string, error) {
	conn, err := tls.Dial("tcp", address, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()
	var b bytes.Buffer
	for _, cert := range conn.ConnectionState().PeerCertificates {
		err := pem.Encode(&b, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		if err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func Run() error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	host, err := normalizeHost(config.Host)
	if err != nil {
		host = config.Host
	}

	// token := config.Token
	// ca := config.CA

	httpClient := &http.Client{}
	if config.Host == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		httpClient = unixHttpClient(filepath.Join(homeDir, ".infra", "infra.sock"))
	}

	p := termenv.ColorProfile()

	app := &cli.App{
		Usage: "manage user & machine access to Kubernetes",
		Commands: []*cli.Command{
			{
				Name:      "login",
				Usage:     "Log in to Infra",
				ArgsUsage: "HOST",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "token",
						Aliases: []string{"t"},
						Usage:   "token to authenticate with",
					},
				},
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

					// TODO: consider case where CA has changed (i.e. cluster certificate rotation, new cluster on same IP)
					ca := config.CA

					// TODO: get CA digest from elsewhere (token, idp, etc)
					if ca == "" {
						insecureClient := &http.Client{
							Transport: &http.Transport{
								TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
							},
						}

						res, err := insecureClient.Get(host + "/v1/ca")
						if err != nil {
							return err
						}

						var response server.RetrieveCAResponse
						err = checkAndDecode(res, &response)
						if err != nil {
							return err
						}

						// Verify certificates manually in case
						opts := x509.VerifyOptions{
							DNSName:       res.TLS.ServerName,
							Intermediates: x509.NewCertPool(),
						}
						for _, cert := range res.TLS.PeerCertificates[1:] {
							opts.Intermediates.AddCert(cert)
						}

						_, err = res.TLS.PeerCertificates[0].Verify(opts)
						if err != nil {
							if _, ok := err.(x509.UnknownAuthorityError); !ok {
								return err
							}

							h := sha256.New()
							h.Write([]byte(response.CA))

							proceed := false
							fmt.Print("Could not verify certificate for cluster ")
							fmt.Print(termenv.String(hostname).Bold())
							fmt.Printf(" (cluster ca fingerprint sha256:%s)\n", base64.URLEncoding.EncodeToString(h.Sum(nil)))
							prompt := &survey.Confirm{
								Message: "Are you sure you want to continue (yes/no)?",
							}

							err := survey.AskOne(prompt, &proceed, survey.WithIcons(func(icons *survey.IconSet) {
								icons.Question.Text = termenv.String("?").Bold().Foreground(p.Color("#0155F9")).String()
							}))
							if err != nil {
								fmt.Println(err.Error())
								return err
							}

							if !proceed {
								return nil
							}

							// write config
							url, err := url.Parse(host)
							if err != nil {
								return err
							}
							config.Host = url.Host
							config.CA = response.CA
							writeConfig(config)

							ca = response.CA
						}
					}

					rootCAs, _ := x509.SystemCertPool()
					if rootCAs == nil {
						rootCAs = x509.NewCertPool()
					}
					rootCAs.AppendCertsFromPEM([]byte(ca))

					client := &http.Client{
						Transport: &http.Transport{
							TLSClientConfig: &tls.Config{
								RootCAs: rootCAs,
							},
						},
					}

					res, err := client.Get(host + "/v1/providers")
					if err != nil {
						return err
					}

					fmt.Println(res.StatusCode)

					form := url.Values{}

					token := ""

					if token == "" {
						res, err := client.Get(host + "/v1/providers")
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

						fmt.Print(termenv.String("✓ ").Bold().Foreground(p.Color("#0057FF")))
						fmt.Println("Logging in with Okta...")
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

					req, err := http.NewRequest("POST", host+"/v1/tokens", strings.NewReader(form.Encode()))
					if err != nil {
						return err
					}

					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

					if token != "" {
						fmt.Print(termenv.String("✓ ").Bold().Foreground(p.Color("#0057FF")))
						fmt.Println("Logging in with Token...")

						// TODO (jmorganca): edit this to use jwt
						req.SetBasicAuth(token, "")
					}

					res, err = client.Do(req)
					if err != nil {
						return err
					}

					var response server.CreateTokenResponse
					err = checkAndDecode(res, &response)
					if err != nil {
						return err
					}

					fmt.Print(termenv.String("✓ ").Bold().Foreground(p.Color("#0057FF")))
					fmt.Println("Logged in...")

					if err = writeConfig(&Config{
						Host:  hostname,
						Token: response.Token,
						CA:    ca,
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
						Server:                   host + "/v1/proxy/",
						CertificateAuthorityData: []byte(ca),
					}
					config.AuthInfos[hostname] = &clientcmdapi.AuthInfo{
						Token: response.Token,
					}
					config.Contexts[hostname] = &clientcmdapi.Context{
						Cluster:  hostname,
						AuthInfo: hostname,
					}
					config.CurrentContext = hostname

					if err = clientcmd.WriteToFile(config, clientcmd.RecommendedHomeFile); err != nil {
						return err
					}

					fmt.Print(termenv.String("✓ ").Bold().Foreground(p.Color("#0057FF")))
					fmt.Println("Kubeconfig updated")

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

							res, err := httpClient.PostForm(host+"/v1/users", form)
							if err != nil {
								return err
							}

							var response server.CreateUserResponse
							err = checkAndDecode(res, &response)
							if err != nil {
								return err
							}

							fmt.Println(response.Email)

							return nil
						},
					},
					{
						Name:    "list",
						Usage:   "list users",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							res, err := httpClient.Get(host + "/v1/users")
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
								rows = append(rows, []string{user.Email, units.HumanDuration(time.Now().UTC().Sub(user.CreatedAt)) + " ago", user.Provider, user.Permission})
							}

							printTable([]string{"EMAIL", "CREATED", "PROVIDER", "PERMISSION"}, rows)

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
				Name:  "server",
				Usage: "Start Infra server",
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
