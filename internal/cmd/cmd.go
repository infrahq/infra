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
	"net/mail"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/cli/browser"
	"github.com/docker/go-units"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/engine"
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
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
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

	if len(header) > 0 {
		table.SetHeader(header)
		table.SetAutoFormatHeaders(true)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	}

	table.SetAutoWrapText(false)
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

func client(host string, token string, skipTlsVerify bool) (client *http.Client, err error) {
	if host == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		return unixClient(filepath.Join(homeDir, ".infra", "infra.sock")), nil
	}

	if skipTlsVerify {
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

func fetchResources() ([]server.Resource, error) {
	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
	if err != nil {
		return nil, err
	}

	serverUrl, err := serverUrl(config.Host)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Get(serverUrl.String() + "/v1/resources")
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []server.Resource `json:"data"`
	}
	err = checkAndDecode(res, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func updateKubeconfig() error {
	resources, err := fetchResources()
	if err != nil {
		return err
	}

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	resourcesJSON, err := json.Marshal(resources)
	if err != nil {
		return err
	}

	// Write destinatinons to json file
	err = os.WriteFile(filepath.Join(home, ".infra", "resources"), resourcesJSON, 0644)
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

	for _, d := range resources {
		kubeConfig.Clusters[d.Name] = &clientcmdapi.Cluster{
			Server:               "https://localhost:32710/client/" + d.Name,
			CertificateAuthority: filepath.Join(home, ".infra", "client", "cert.pem"),
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[d.Name] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:    executable,
				Args:       []string{"creds"},
				APIVersion: "client.authentication.k8s.io/v1alpha1",
			},
		}
		kubeConfig.Contexts[d.Name] = &clientcmdapi.Context{
			Cluster:  d.Name,
			AuthInfo: d.Name,
		}
	}

	for name, c := range kubeConfig.Clusters {
		if !strings.HasPrefix(c.Server, "https://localhost:32710/client/") {
			continue
		}

		var exists bool
		for _, r := range resources {
			if name == r.Name {
				exists = true
			}
		}

		if !exists {
			delete(kubeConfig.Clusters, name)
			delete(kubeConfig.Contexts, name)
			delete(kubeConfig.AuthInfos, name)
		}
	}

	if err = clientcmd.WriteToFile(kubeConfig, clientcmd.RecommendedHomeFile); err != nil {
		return err
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:   "infra",
	Short: "Infrastructure Identity & Access Management (IAM)",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

func newLoginCmd() *cobra.Command {
	var email, password string

	cmd := &cobra.Command{
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
			skipTlsVerify, err := cmd.PersistentFlags().GetBool("skip-tls-verify")
			if err != nil {
				return err
			}

			httpClient, err := client(host, "", skipTlsVerify)
			if err != nil {
				return err
			}

			res, err := httpClient.Get(host + "/v1/providers")
			if err != nil {
				if strings.Contains(err.Error(), "x509: certificate signed by unknown authority") {
					return errors.New(err.Error() + "\n" + "Use \"infra login " + host + " -k or --skip-tls-verify\" to bypass CA verification for all future requests")
				}
				return err
			}

			var response struct{ Data []server.Provider }
			if err = checkAndDecode(res, &response); err != nil {
				return err
			}

			sort.Slice(response.Data, func(i, j int) bool {
				return response.Data[i].Created > response.Data[j].Created
			})

			form := url.Values{}

			if len(email) > 0 && len(password) > 0 {
				hasInfra := false
				for _, p := range response.Data {
					if p.Kind == "infra" {
						hasInfra = true
					}
				}

				if !hasInfra {
					return errors.New("user & password flags provided but infra provider is not enabled")
				}

				form.Add("email", email)
				form.Add("password", password)

				fmt.Println(blue("✓") + " Logging in with username & password...")
			} else {
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
				Host:          host,
				Token:         loginResponse.Token,
				SkipTLSVerify: skipTlsVerify,
			}

			err = writeConfig(config)
			if err != nil {
				return err
			}

			// Generate client certs
			home, err := homedir.Dir()
			if err != nil {
				return err
			}

			if err = os.MkdirAll(filepath.Join(home, ".infra", "client"), os.ModePerm); err != nil {
				return err
			}

			if _, err := os.Stat(filepath.Join(home, ".infra", "client", "cert.pem")); os.IsNotExist(err) {
				certBytes, keyBytes, err := certs.GenerateSelfSignedCert([]string{"localhost", "localhost:32710"})
				if err != nil {
					return err
				}

				if err = ioutil.WriteFile(filepath.Join(home, ".infra", "client", "cert.pem"), certBytes, 0644); err != nil {
					return err
				}

				if err = ioutil.WriteFile(filepath.Join(home, ".infra", "client", "key.pem"), keyBytes, 0644); err != nil {
					return err
				}

				// Kill client
				contents, err := ioutil.ReadFile(filepath.Join(home, ".infra", "client", "pid"))
				if err != nil && !os.IsNotExist(err) {
					return err
				}

				var pid int
				if !os.IsNotExist(err) {
					pid, err = strconv.Atoi(string(contents))
					if err != nil {
						return err
					}

					process, _ := os.FindProcess(int(pid))
					process.Kill()
				}

				os.Remove(filepath.Join(home, ".infra", "client", "pid"))
			}

			err = updateKubeconfig()
			if err != nil {
				return err
			}

			fmt.Println(blue("✓") + " Kubeconfig updated")

			return nil
		},
	}

	cmd.PersistentFlags().BoolP("skip-tls-verify", "k", false, "skip TLS verification")
	cmd.Flags().StringVarP(&email, "user", "u", "", "user email")
	cmd.Flags().StringVarP(&password, "password", "p", "", "user password")

	return cmd
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List clusters",
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, err := fetchResources()
		if err != nil {
			return err
		}

		sort.Slice(resources, func(i, j int) bool {
			return resources[i].Created > resources[j].Created
		})

		kubeConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()

		rows := [][]string{}
		for _, dest := range resources {
			star := ""
			if dest.Name == kubeConfig.CurrentContext {
				star = "*"
			}
			rows = append(rows, []string{dest.Name + star, dest.KubernetesEndpoint})
		}

		printTable([]string{"NAME", "ENDPOINT"}, rows)
		fmt.Println()
		fmt.Println("To connect, run \"kubectl config use-context <name>\"")
		fmt.Println()

		err = updateKubeconfig()
		if err != nil {
			return err
		}

		return nil
	},
}

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create kubectl command
			config, err := readConfig()
			if err != nil {
				return err
			}

			httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
			if err != nil {
				return err
			}

			serverUrl, err := serverUrl(config.Host)
			if err != nil {
				return err
			}

			res, err := httpClient.Get(serverUrl.String() + "/v1/apikeys")
			if err != nil {
				return err
			}

			var response struct {
				Data []server.APIKey `json:"data"`
			}
			err = checkAndDecode(res, &response)
			if err != nil {
				return err
			}

			if len(response.Data) == 0 {
				return errors.New("no valid api keys")
			}

			fmt.Println()
			fmt.Println("To connect your cluster via kubectl, run:")
			fmt.Println()
			fmt.Println("kubectl create namespace infra")
			fmt.Print("kubectl create configmap infra-engine --from-literal='name=" + args[0] + "' --from-literal='server=" + config.Host + "'")
			if config.SkipTLSVerify {
				fmt.Print(" --from-literal='skip-tls-verify=1'")
			}
			fmt.Println(" --namespace=infra")
			fmt.Println("kubectl create secret generic infra-engine --from-literal='api-key=" + response.Data[0].Key + "' --namespace=infra")
			fmt.Println("kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/main/deploy/engine.yaml")

			return nil
		},
	}

	return cmd
}

func newGrantCmd() *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:     "grant USER RESOURCE",
		Short:   "Grant access to a user",
		Example: "$ infra grant user@example.com production --role kubernetes.editor",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := readConfig()
			if err != nil {
				return err
			}

			httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
			if err != nil {
				return err
			}

			serverUrl, err := serverUrl(config.Host)
			if err != nil {
				return err
			}

			form := url.Values{}
			form.Add("user", args[0])
			form.Add("resource", args[1])

			if role != "" {
				form.Add("role", role)
			}

			res, err := httpClient.PostForm(serverUrl.String()+"/v1/grants", form)
			if err != nil {
				return err
			}

			var grant server.Grant
			err = checkAndDecode(res, &grant)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&role, "role", "r", "", "role")

	return cmd
}

func newRevokeCmd() *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "revoke USER RESOURCE",
		Short: "Revoke access from a user",
		Example: heredoc.Doc(`
			$ infra revoke user@example.com production
			$ infra revoke user@example.com production --role kubernetes.editor`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := readConfig()
			if err != nil {
				return err
			}

			httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
			if err != nil {
				return err
			}

			serverUrl, err := serverUrl(config.Host)
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Add("user", args[0])
			params.Add("resource", args[1])

			if role != "" {
				params.Add("role", role)
			}

			res, err := httpClient.Get(serverUrl.String() + "/v1/grants?" + params.Encode())
			if err != nil {
				return err
			}

			var response struct {
				Data []server.Grant `json:"data"`
			}
			err = checkAndDecode(res, &response)
			if err != nil {
				return err
			}

			for _, g := range response.Data {
				req, err := http.NewRequest(http.MethodDelete, serverUrl.String()+"/v1/grants/"+g.ID, nil)
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
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&role, "role", "r", "", "role")

	return cmd
}

var inspectCmd = &cobra.Command{
	Use:   "inspect CLUSTER|USER",
	Short: "Inspect access",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		serverUrl, err := serverUrl(config.Host)
		if err != nil {
			return err
		}

		// TODO (jmorganca): actually check server side when we have
		// groups or non-email users
		_, err = mail.ParseAddress(args[0])
		isUser := err == nil

		params := url.Values{}

		if isUser {
			params.Add("user", args[0])
		} else {
			params.Add("resource", args[0])
		}

		res, err := httpClient.Get(serverUrl.String() + "/v1/grants?" + params.Encode())
		if err != nil {
			return err
		}

		var response struct {
			Data []server.Grant `json:"data"`
		}
		err = checkAndDecode(res, &response)
		if err != nil {
			return err
		}

		sort.Slice(response.Data, func(i, j int) bool {
			return response.Data[i].Created > response.Data[j].Created
		})

		rows := [][]string{}
		for _, grant := range response.Data {
			if isUser {
				rows = append(rows, []string{grant.Resource.Name, grant.Role.Name})
			} else {
				rows = append(rows, []string{grant.User.Email, grant.Role.Name})
			}
		}

		if isUser {
			printTable([]string{"RESOURCE", "ROLE"}, rows)
		} else {
			printTable([]string{"USER", "ROLE"}, rows)
		}

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

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
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
	Use:   "delete USER",
	Short: "delete a user",
	Args:  cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra users delete user@example.com`),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		serverUrl, err := serverUrl(config.Host)
		if err != nil {
			return err
		}

		params := url.Values{}
		params.Set("email", args[0])

		res, err := httpClient.Get(serverUrl.String() + "/v1/users?" + params.Encode())
		if err != nil {
			return err
		}

		var listResponse struct {
			Data []server.User `json:"data"`
		}
		err = checkAndDecode(res, &listResponse)
		if err != nil {
			return err
		}

		for _, u := range listResponse.Data {
			req, err := http.NewRequest(http.MethodDelete, serverUrl.String()+"/v1/users/"+u.ID, nil)
			if err != nil {
				log.Fatal(err)
			}

			res, err := httpClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			var response server.DeleteResponse
			err = checkAndDecode(res, &response)
			if err != nil {
				return err
			}

			res.Body.Close()
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

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
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
			providers := ""
			for i, p := range user.Providers {
				if i > 0 {
					providers += ","
				}
				providers += p.Kind
			}

			infraGrant := ""
			for _, g := range user.Grants {
				if g.Resource.Name == "infra" {
					infraGrant = g.Role.Name
				}
			}

			rows = append(rows, []string{user.Email, providers, units.HumanDuration(time.Now().UTC().Sub(time.Unix(user.Created, 0))) + " ago", infraGrant})
		}

		printTable([]string{"EMAIL", "PROVIDERS", "CREATED", "ROLE"}, rows)

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

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
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
			rows = append(rows, []string{provider.ID, provider.Kind, units.HumanDuration(time.Now().UTC().Sub(time.Unix(provider.Created, 0))) + " ago", info})
		}

		printTable([]string{"PROVIDER ID", "KIND", "CREATED", "DESCRIPTION"}, rows)

		return nil
	},
}

func newprovidersCreateCmd() *cobra.Command {
	var apiToken, domain, clientID, clientSecret string

	cmd := &cobra.Command{
		Use:     "create KIND",
		Aliases: []string{"add"},
		Short:   "Create a provider connection",
		Args:    cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			$ infra providers create okta \
				--domain example.okta.com \
				--apiToken 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd \
				--clientID 0oapn0qwiQPiMIyR35d6 \
				--clientSecret jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2`),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := readConfig()
			if err != nil {
				return err
			}

			httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
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

func logout() error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
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

	return nil
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of Infra server",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := logout()
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

		for name, c := range kubeConfig.Clusters {
			if strings.HasPrefix(c.Server, "https://localhost:32710/client/") {
				delete(kubeConfig.Clusters, name)
				delete(kubeConfig.Contexts, name)
				delete(kubeConfig.AuthInfos, name)
			}
		}

		if len(kubeConfig.Contexts) == 0 {
			os.Remove(clientcmd.RecommendedHomeFile)
		} else {
			_, ok := kubeConfig.Contexts[kubeConfig.CurrentContext]
			if !ok {
				var firstName string
				for name := range kubeConfig.Contexts {
					firstName = name
					break
				}
				kubeConfig.CurrentContext = firstName
			}
		}

		if err = clientcmd.WriteToFile(kubeConfig, clientcmd.RecommendedHomeFile); err != nil {
			return err
		}
		return nil
	},
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

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
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

func newEngineCmd() *cobra.Command {
	var options engine.Options

	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Start Infra engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.Server == "" {
				return errors.New("server not specified (--server or INFRA_ENGINE_SERVER)")
			}
			if options.Name == "" {
				return errors.New("name not specified (--name or INFRA_ENGINE_NAME)")
			}
			if options.APIKey == "" {
				return errors.New("api-key not specified (--api-key or INFRA_ENGINE_API_KEY)")
			}
			return engine.Run(options)
		},
	}

	cmd.PersistentFlags().BoolVarP(&options.SkipTLSVerify, "skip-tls-verify", "k", len(os.Getenv("INFRA_ENGINE_SKIP_TLS_VERIFY")) > 0, "skip TLS verification")
	cmd.Flags().StringVarP(&options.Server, "server", "s", os.Getenv("INFRA_ENGINE_SERVER"), "server hostname")
	cmd.Flags().StringVarP(&options.Name, "name", "n", os.Getenv("INFRA_ENGINE_NAME"), "cluster name")
	cmd.Flags().StringVar(&options.APIKey, "api-key", os.Getenv("INFRA_ENGINE_API_KEY"), "api key")

	return cmd
}

var credsCmd = &cobra.Command{
	Use:    "creds",
	Short:  "Generate credentials",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO (jmorganca): First try to read cached token
		// TODO (jmorganca): this will need to change to multiple files with multiple cluster support
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
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

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		if err = os.MkdirAll(filepath.Join(home, ".infra", "cache"), os.ModePerm); err != nil {
			return err
		}

		contents, err := ioutil.ReadFile(filepath.Join(home, ".infra", "client", "pid"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		var pid int
		// see if proxy process is already running
		if !os.IsNotExist(err) {
			pid, err = strconv.Atoi(string(contents))
			if err != nil {
				return err
			}

			// verify process is still running
			process, err := os.FindProcess(int(pid))
			if process == nil || err != nil {
				pid = 0
			}

			err = process.Signal(syscall.Signal(0))
			if err != nil {
				pid = 0
			}
		}

		if pid == 0 {
			os.Remove(filepath.Join(home, ".infra", "client", "pid"))

			cmd := exec.Command(os.Args[0], "client")
			err = cmd.Start()
			if err != nil {
				return err
			}

			for {
				if _, err := os.Stat(filepath.Join(home, ".infra", "client", "pid")); os.IsNotExist(err) {
					time.Sleep(25 * time.Millisecond)
				} else {
					break
				}
			}
		}

		fmt.Println(string(bts))

		return nil
	},
}

var clientCmd = &cobra.Command{
	Use:    "client",
	Short:  "Run local client to relay requests",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting client")
		return RunLocalClient()
	},
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newGrantCmd())
	rootCmd.AddCommand(newRevokeCmd())
	rootCmd.AddCommand(inspectCmd)

	usersCmd.AddCommand(usersCreateCmd)
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersDeleteCmd)
	rootCmd.AddCommand(usersCmd)

	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(newprovidersCreateCmd())
	providersCmd.AddCommand(providersDeleteCmd)
	rootCmd.AddCommand(providersCmd)

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(logoutCmd)

	serverCmd, err := newServerCmd()
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(newEngineCmd())

	// Hidden commands
	rootCmd.AddCommand(credsCmd)
	rootCmd.AddCommand(clientCmd)

	return rootCmd, nil
}

func Run() error {
	cmd, err := NewRootCmd()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
