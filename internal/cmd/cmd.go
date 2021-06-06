package cmd

import (
	"context"
	"crypto/tls"
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
	"sort"
	"strconv"
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
	"gopkg.in/square/go-jose.v2/jwt"
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

		var response struct{ Data []server.Provider }
		if err = checkAndDecode(res, &response); err != nil {
			return err
		}

		sort.Slice(response.Data, func(i, j int) bool {
			return response.Data[i].Created > response.Data[j].Created
		})

		options := []string{}
		for _, p := range response.Data {
			if p.Kind == "okta" {
				options = append(options, fmt.Sprintf("Okta [%s]", p.Domain))
			} else if p.Kind == "infra" {
				options = append(options, "Username & password")
			} else {
				options = append(options, p.Kind)
			}
		}

		var option int
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
		}

		form := url.Values{}

		switch {
		// Okta
		case response.Data[option].Kind == "okta":
			// Start OIDC flow
			// Get auth code from Okta
			// Send auth code to Infra to log in as a user
			state := generate.RandString(12)
			authorizeUrl := "https://" + response.Data[option].Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + response.Data[option].ClientID + "&response_type=code&scope=openid+email&nonce=" + generate.RandString(10) + "&state=" + state

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

			form.Add("okta-domain", response.Data[option].Domain)
			form.Add("okta-code", code)

		case response.Data[option].Kind == "infra":
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

		var tokenResponse struct{ Token string }
		err = checkAndDecode(res, &tokenResponse)
		if err != nil {
			return err
		}

		fmt.Println(blue("✓") + " Logged in...")

		config := &Config{
			Host:     host,
			Token:    tokenResponse.Token,
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
			Token: tokenResponse.Token,
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

		var user server.User
		err = checkAndDecode(res, &user)
		if err != nil {
			return err
		}

		fmt.Println(user.ID)

		return nil
	},
}

var usersDeleteCmd = &cobra.Command{
	Use:   "delete ID",
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

		fmt.Println(user)
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

		var response struct{ Data []server.User }
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
			roles := ""
			for i, p := range user.Permissions {
				if i > 0 {
					roles += ","
				}
				roles += p.Role.Name
			}
			providers := ""
			for i, p := range user.Providers {
				if i > 0 {
					providers += ","
				}
				providers += p.Kind
			}
			rows = append(rows, []string{user.ID, user.Email + star, providers, units.HumanDuration(time.Now().UTC().Sub(time.Unix(user.Created, 0))) + " ago", roles})
		}

		printTable([]string{"ID", "EMAIL", "PROVIDERS", "CREATED", "ROLES"}, rows)

		return nil
	},
}

var providersCmd = &cobra.Command{
	Use:     "providers",
	Aliases: []string{"provider"},
	Short:   "Manage identity providers",
}

var providersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List providers",
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

		res, err := httpClient.Get(serverUrl.String() + "/v1/providers")
		if err != nil {
			return err
		}

		var response struct{ Data []server.Provider }
		err = checkAndDecode(res, &response)
		if err != nil {
			return err
		}

		sort.Slice(response.Data, func(i, j int) bool {
			return response.Data[i].Created > response.Data[j].Created
		})

		rows := [][]string{}
		for _, provider := range response.Data {
			info := ""
			switch provider.Kind {
			case "okta":
				info = provider.Domain
			case "infra":
				info = "Built-in provider"
			}
			rows = append(rows, []string{provider.ID, provider.Kind, strconv.Itoa(len(provider.Users)), units.HumanDuration(time.Now().UTC().Sub(time.Unix(provider.Created, 0))) + " ago", info})
		}

		printTable([]string{"ID", "KIND", "USERS", "CREATED", "DESCRIPTION"}, rows)

		return nil
	},
}

func newProvidersCreateCmd() *cobra.Command {
	var apiToken, domain, clientID, clientSecret string

	cmd := &cobra.Command{
		Use:     "create KIND",
		Aliases: []string{"add"},
		Short:   "Create a provider connection",
		Args:    cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			$ infra providers create okta --domain example.okta.com \
				--apiToken 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd \
				--clientID 0oapn0qwiQPiMIyR35d6 \
				--clientSecret jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2`),
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

			form := url.Values{}
			form.Add("kind", args[0])
			form.Add("apiToken", apiToken)
			form.Add("domain", domain)
			form.Add("clientID", clientID)
			form.Add("clientSecret", clientSecret)

			res, err := httpClient.PostForm(serverUrl.String()+"/v1/providers", form)
			if err != nil {
				return err
			}

			var provider server.Provider
			err = checkAndDecode(res, &provider)
			if err != nil {
				return err
			}

			fmt.Println(provider.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&apiToken, "api-token", "", "Api Token")
	cmd.Flags().StringVar(&domain, "domain", "", "Identity provider domain (e.g. example.okta.com)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID for single sign on")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Client Secret for single sign on")

	return cmd
}

var providersDeleteCmd = &cobra.Command{
	Use:     "delete ID",
	Aliases: []string{"rm"},
	Short:   "Delete a provider connection",
	Args:    cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra providers delete n7bha2pxjpa01a`),
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

		id := args[0]
		req, err := http.NewRequest(http.MethodDelete, serverUrl.String()+"/v1/providers/"+id, nil)
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

		fmt.Println(id)

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

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(loginCmd)
	loginCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")

	usersCmd.AddCommand(usersCreateCmd)
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersDeleteCmd)
	usersCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")

	rootCmd.AddCommand(usersCmd)

	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(newProvidersCreateCmd())
	providersCmd.AddCommand(providersDeleteCmd)
	providersCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")
	rootCmd.AddCommand(providersCmd)

	serverCmd, err := newServerCmd()
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(serverCmd)

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
