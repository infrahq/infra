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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1alpha1 "k8s.io/client-go/pkg/apis/clientauthentication/v1alpha1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Config struct {
	Host     string `json:"host"`
	Token    string `json:"token"`
	Insecure bool   `json:"insecure,omitempty"`
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

func removeConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(homeDir, ".infra", "config"))
	if err != nil {
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
				Message: "Choose a login source",
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

		req, err := http.NewRequest("POST", host+"/v1/login", strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		res, err = httpClient.Do(req)
		if err != nil {
			return err
		}

		var loginResponse struct {
			Token string `json:"token"`
		}
		err = checkAndDecode(res, &loginResponse)
		if err != nil {
			return err
		}

		fmt.Println(blue("✓") + " Logged in...")

		config := &Config{
			Host:  host,
			Token: loginResponse.Token,
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

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		execArgs := []string{"creds"}
		if insecure {
			execArgs = append(execArgs, "--insecure")
		}

		kubeConfig.AuthInfos[hostname] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:    executable,
				Args:       execArgs,
				APIVersion: "client.authentication.k8s.io/v1alpha1",
			},
		}
		kubeConfig.Contexts[hostname] = &clientcmdapi.Context{
			Cluster:  hostname,
			AuthInfo: hostname,
		}
		kubeConfig.CurrentContext = hostname

		if err = clientcmd.WriteToFile(kubeConfig, clientcmd.RecommendedHomeFile); err != nil {
			return err
		}

		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		os.Remove(filepath.Join(home, ".infra", "cache", "kubectl-token"))

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

		insecure, _ := cmd.Flags().GetBool("insecure")
		httpClient, err := client(config.Host, config.Token, insecure)
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

		insecure, _ := cmd.Flags().GetBool("insecure")
		httpClient, err := client(config.Host, config.Token, insecure)
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

		insecure, _ := cmd.Flags().GetBool("insecure")
		httpClient, err := client(config.Host, config.Token, insecure)
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

		var response struct {
			Data []server.User `json:"data"`
		}
		err = checkAndDecode(res, &response)
		if err != nil {
			return err
		}

		sort.Slice(response.Data, func(i, j int) bool {
			return response.Data[i].Created > response.Data[j].Created
		})

		rows := [][]string{}
		for _, user := range response.Data {
			roles := ""
			for i, p := range user.Permissions {
				if i > 0 {
					roles += ","
				}
				roles += p.Role.Name
			}
			sources := ""
			for i, p := range user.Providers {
				if i > 0 {
					sources += ","
				}
				sources += p.Kind
			}
			rows = append(rows, []string{user.ID, user.Email, sources, units.HumanDuration(time.Now().UTC().Sub(time.Unix(user.Created, 0))) + " ago", roles})
		}

		printTable([]string{"USER ID", "EMAIL", "sourceS", "CREATED", "ROLES"}, rows)

		return nil
	},
}

var sourcesCmd = &cobra.Command{
	Use:     "sources",
	Aliases: []string{"source"},
	Short:   "Manage identity sources",
}

var sourcesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		insecure, _ := cmd.Flags().GetBool("insecure")
		httpClient, err := client(config.Host, config.Token, insecure)
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
		for _, source := range response.Data {
			info := ""
			switch source.Kind {
			case "okta":
				info = source.Domain
			case "infra":
				info = "Built-in source"
			}
			rows = append(rows, []string{source.ID, source.Kind, strconv.Itoa(len(source.Users)), units.HumanDuration(time.Now().UTC().Sub(time.Unix(source.Created, 0))) + " ago", info})
		}

		printTable([]string{"SOURCE ID", "KIND", "USERS", "CREATED", "DESCRIPTION"}, rows)

		return nil
	},
}

func newsourcesCreateCmd() *cobra.Command {
	var apiToken, domain, clientID, clientSecret string

	cmd := &cobra.Command{
		Use:     "create KIND",
		Aliases: []string{"add"},
		Short:   "Create a source connection",
		Args:    cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			$ infra sources create okta \
				--domain example.okta.com \
				--apiToken 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd \
				--clientID 0oapn0qwiQPiMIyR35d6 \
				--clientSecret jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2`),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := readConfig()
			if err != nil {
				return err
			}

			insecure, _ := cmd.Flags().GetBool("insecure")
			httpClient, err := client(config.Host, config.Token, insecure)
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

			var source server.Provider
			err = checkAndDecode(res, &source)
			if err != nil {
				return err
			}

			fmt.Println(source.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&apiToken, "api-token", "", "Api Token")
	cmd.Flags().StringVar(&domain, "domain", "", "Identity source domain (e.g. example.okta.com)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID for single sign on")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Client Secret for single sign on")

	return cmd
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of Infra server",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		insecure, _ := cmd.Flags().GetBool("insecure")
		httpClient, err := client(config.Host, config.Token, insecure)
		if err != nil {
			return err
		}

		serverUrl, err := serverUrl(config.Host)
		if err != nil {
			return err
		}

		if config.Token == "" {
			return nil
		}

		_, err = httpClient.Post(serverUrl.String()+"/v1/logout", "application/x-www-form-urlencoded", nil)
		if err != nil {
			return err
		}

		err = removeConfig()
		if err != nil {
			return err
		}

		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		os.Remove(filepath.Join(home, ".infra", "cache", "kubectl-token"))

		return nil
	},
}

var sourcesDeleteCmd = &cobra.Command{
	Use:     "delete ID",
	Aliases: []string{"rm"},
	Short:   "Delete a source connection",
	Args:    cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra sources delete n7bha2pxjpa01a`),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		insecure, _ := cmd.Flags().GetBool("insecure")
		httpClient, err := client(config.Host, config.Token, insecure)
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

var credsCmd = &cobra.Command{
	Use:    "creds",
	Short:  "Generate credentials",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// First try to read cached token
		// TODO (jmorganca): this will need to change to multiple files with multiple cluster support
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "cache", "kubectl-token"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		if len(contents) > 0 {
			var cached clientauthenticationv1alpha1.ExecCredential
			err := json.Unmarshal(contents, &cached)
			if err == nil {
				if time.Now().Before(cached.Status.ExpirationTimestamp.Time) {
					fmt.Println(string(contents))
					return nil
				} else {
					err = os.Remove(filepath.Join(homeDir, ".infra", "cache", "kubectl-token"))
					if err != nil {
						return err
					}
				}
			}
		}

		config, err := readConfig()
		if err != nil {
			return err
		}

		insecure, _ := cmd.Flags().GetBool("insecure")
		httpClient, err := client(config.Host, config.Token, insecure)
		if err != nil {
			return err
		}

		serverUrl, err := serverUrl(config.Host)
		if err != nil {
			return err
		}

		if config.Token == "" {
			return nil
		}

		res, err := httpClient.Post(serverUrl.String()+"/v1/creds", "application/x-www-form-urlencoded", nil)
		if err != nil {
			return err
		}

		var response struct {
			Token               string `json:"token"`
			ExpirationTimestamp string `json:"expirationTimestamp"`
		}
		err = checkAndDecode(res, &response)
		if err != nil {
			return err
		}

		expiry, err := time.Parse(time.RFC3339, response.ExpirationTimestamp)
		if err != nil {
			return err
		}

		execCredential := &clientauthenticationv1alpha1.ExecCredential{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ExecCredential",
				APIVersion: clientauthenticationv1alpha1.SchemeGroupVersion.String(),
			},
			Spec: clientauthenticationv1alpha1.ExecCredentialSpec{},
			Status: &clientauthenticationv1alpha1.ExecCredentialStatus{
				Token:               response.Token,
				ExpirationTimestamp: &metav1.Time{Time: expiry},
			},
		}

		bts, err := json.Marshal(execCredential)
		if err != nil {
			return err
		}

		if err = os.MkdirAll(filepath.Join(homeDir, ".infra", "cache"), os.ModePerm); err != nil {
			return err
		}

		if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "cache", "kubectl-token"), []byte(bts), 0644); err != nil {
			return err
		}

		fmt.Println(string(bts))

		return nil
	},
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	usersCmd.AddCommand(usersCreateCmd)
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersDeleteCmd)
	usersCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")

	rootCmd.AddCommand(usersCmd)

	sourcesCmd.AddCommand(sourcesListCmd)
	sourcesCmd.AddCommand(newsourcesCreateCmd())
	sourcesCmd.AddCommand(sourcesDeleteCmd)
	sourcesCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")
	rootCmd.AddCommand(sourcesCmd)

	rootCmd.AddCommand(loginCmd)
	loginCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")
	rootCmd.AddCommand(logoutCmd)

	rootCmd.AddCommand(credsCmd)
	credsCmd.PersistentFlags().BoolP("insecure", "i", false, "skip TLS verification")

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
