package cmd

import (
	"context"
	"crypto/tls"
	"log"
	"sort"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/cli/browser"
	"github.com/docker/go-units"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server"
	"github.com/mitchellh/go-homedir"
	"github.com/muesli/termenv"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/square/go-jose/jwt"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Config struct {
	Host     string `yaml:"host"`
	Token    string `yaml:"token"`
	Insecure bool   `yaml:"insecure,omitempty"`
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

func blue(s string) string {
	return termenv.String(s).Bold().Foreground(termenv.ColorProfile().Color("#0057FF")).String()
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

func unixClient(path string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", path)
			},
		},
	}
}

func serverUrl(host string) (*url.URL, error) {
	if host == "" {
		return urlx.Parse("http://unix")
	}

	return serverUrlFromString(host)
}

func serverUrlFromString(host string) (*url.URL, error) {
	u, err := urlx.Parse(host)
	if err != nil {
		return nil, err
	}

	u.Scheme = "https"

	return u, nil
}

func client(host string, token string, insecure bool) (client *http.Client, err error) {
	if host == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		return unixClient(filepath.Join(homeDir, ".infra", "infra.sock")), nil
	}

	if insecure {
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
	}

	return &http.Client{
		Transport: &TokenTransport{
			Token:     token,
			Transport: http.DefaultTransport,
		},
	}, nil
}

var rootCmd = &cobra.Command{
	Use:   "infra",
	Short: "Infra – identity & access management for infrastructure",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
	Example: heredoc.Doc(`
		# Log into an Infra server
		$ infra login infra.example.com

		# Create a user
		$ infra users create test@test.com p4ssw0rd

		# List users
		$ infra users ls

		# Delete a user
		$ infra users delete test@test.com
		`),
}

var loginCmd = &cobra.Command{
	Use:     "login HOST",
	Short:   "Log in to Infra server",
	Args:    cobra.ExactArgs(1),
	Example: "$ infra login infra.example.com",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverUrl, err := serverUrlFromString(args[0])
		if err != nil {
			return err
		}

		host := serverUrl.String()
		insecure, err := cmd.PersistentFlags().GetBool("insecure")
		if err != nil {
			return err
		}

		httpClient, err := client(host, "", insecure)
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
		okta := fmt.Sprintf("Okta [%s]", response.OktaDomain)
		userpass := "Username & password"

		options := []string{}
		if response.OktaClientID != "" && response.OktaDomain != "" {
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
			authorizeUrl := "https://" + response.OktaDomain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + response.OktaClientID + "&response_type=code&scope=openid+email&nonce=" + generate.RandString(10) + "&state=" + state

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

		config := &Config{
			Host:     host,
			Token:    createTokenResponse.Token,
			Insecure: insecure,
		}

		err = writeConfig(config)
		if err != nil {
			return err
		}

		// Load default config and merge new config in
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		defaultConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		kubeConfig, err := defaultConfig.RawConfig()
		if err != nil {
			return err
		}

		hostname := serverUrl.Hostname()

		kubeConfig.Clusters[hostname] = &clientcmdapi.Cluster{
			Server: serverUrl.String() + "/v1/proxy",
		}

		if insecure {
			kubeConfig.Clusters[hostname].InsecureSkipTLSVerify = true
		}

		kubeConfig.AuthInfos[hostname] = &clientcmdapi.AuthInfo{
			Token: createTokenResponse.Token,
		}
		kubeConfig.Contexts[hostname] = &clientcmdapi.Context{
			Cluster:  hostname,
			AuthInfo: hostname,
		}
		kubeConfig.CurrentContext = hostname

		if err = clientcmd.WriteToFile(kubeConfig, clientcmd.RecommendedHomeFile); err != nil {
			return err
		}

		fmt.Println(blue("✓") + " Kubeconfig updated")

		return nil
	},
}

var usersCmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user"},
	Short:   "Manage users",
}

var usersCreateCmd = &cobra.Command{
	Use:     "create EMAIL PASSWORD",
	Short:   "create a user",
	Args:    cobra.ExactArgs(2),
	Example: "$ infra users create admin@example.com p4assw0rd",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.Insecure)
		if err != nil {
			return err
		}

		serverUrl, err := serverUrl(config.Host)
		if err != nil {
			return err
		}

		email := args[0]
		password := args[1]
		form := url.Values{}
		form.Add("email", email)
		form.Add("password", password)

		res, err := httpClient.PostForm(serverUrl.String()+"/v1/users", form)
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
}

var usersDeleteCmd = &cobra.Command{
	Use:   "delete EMAIL",
	Short: "delete a user",
	Args:  cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra users delete user@example.com`),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.Insecure)
		if err != nil {
			return err
		}

		serverUrl, err := serverUrl(config.Host)
		if err != nil {
			return err
		}

		user := args[0]
		req, err := http.NewRequest(http.MethodDelete, serverUrl.String()+"/v1/users/"+user, nil)
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
		return nil
	},
}

var usersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List users",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.Insecure)
		if err != nil {
			return err
		}

		serverUrl, err := serverUrl(config.Host)
		if err != nil {
			return err
		}

		res, err := httpClient.Get(serverUrl.String() + "/v1/users")
		if err != nil {
			return err
		}

		var response server.ListUsersResponse
		err = checkAndDecode(res, &response)
		if err != nil {
			return err
		}

		email := ""
		if config.Token != "" {
			tok, err := jwt.ParseSigned(config.Token)
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
			rows = append(rows, []string{user.Email + star, user.Provider, user.Permission.Name, units.HumanDuration(time.Now().UTC().Sub(time.Unix(user.Created, 0))) + " ago"})
		}

		printTable([]string{"EMAIL", "PROVIDER", "PERMISSION", "CREATED"}, rows)

		return nil
	},
}

func newServerCmd() (*cobra.Command, error) {
	var options server.Options

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start Infra server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Run(options)
		},
	}

	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	serverCmd.Flags().StringVarP(&options.ConfigPath, "config", "c", "", "server config file")
	serverCmd.Flags().StringVar(&options.DBPath, "db", filepath.Join(home, ".infra", "infra.db"), "path to database file")
	serverCmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(home, ".infra", "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")
	serverCmd.Flags().BoolVar(&options.UI, "ui", false, "enable experimental UI")
	serverCmd.Flags().BoolVar(&options.UIProxy, "ui-proxy", false, "proxy ui requests to localhost:3000")

	return serverCmd, nil
}

func Run() error {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(loginCmd)
	loginCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")

	usersCmd.AddCommand(usersCreateCmd)
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersDeleteCmd)
	usersCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")

	rootCmd.AddCommand(usersCmd)

	serverCmd, err := newServerCmd()
	if err != nil {
		return err
	}

	rootCmd.AddCommand(serverCmd)

	return rootCmd.Execute()
}
