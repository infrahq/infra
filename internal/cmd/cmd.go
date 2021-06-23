package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
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
	"github.com/infrahq/infra/internal/registry"
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

	er := &registry.ErrorResponse{}
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

func registryUrl(host string) (*url.URL, error) {
	if host == "" {
		return urlx.Parse("http://unix")
	}

	return registryUrlFromString(host)
}

func registryUrlFromString(host string) (*url.URL, error) {
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

func fetchDestinations() ([]registry.Destination, error) {
	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
	if err != nil {
		return nil, err
	}

	registryUrl, err := registryUrl(config.Host)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Get(registryUrl.String() + "/v1/destinations")
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []registry.Destination `json:"data"`
	}
	err = checkAndDecode(res, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func updateKubeconfig() (bool, error) {
	destinations, err := fetchDestinations()
	if err != nil {
		return false, err
	}

	home, err := homedir.Dir()
	if err != nil {
		return false, err
	}

	destinationsJSON, err := json.Marshal(destinations)
	if err != nil {
		return false, err
	}

	// Write destinatinons to json file
	err = os.WriteFile(filepath.Join(home, ".infra", "destinations"), destinationsJSON, 0644)
	if err != nil {
		return false, err
	}

	// Load default config and merge new config in
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	defaultConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return false, err
	}

	for _, d := range destinations {
		kubeConfig.Clusters[d.Name] = &clientcmdapi.Cluster{
			Server:               "https://localhost:32710/client/" + d.Name,
			CertificateAuthority: filepath.Join(home, ".infra", "client", "cert.pem"),
		}

		executable, err := os.Executable()
		if err != nil {
			return false, err
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
		for _, d := range destinations {
			if name == d.Name {
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
		return false, err
	}

	return len(destinations) > 0, nil
}

var rootCmd = &cobra.Command{
	Use:   "infra",
	Short: "Infrastructure Identity & Access Management (IAM)",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

func newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login REGISTRY",
		Short:   "Log in to an Infra Registry",
		Args:    cobra.ExactArgs(1),
		Example: "$ infra login infra.example.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			registryUrl, err := registryUrlFromString(args[0])
			if err != nil {
				return err
			}

			host := registryUrl.String()
			if err != nil {
				return err
			}

			insecureClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}

			res, err := insecureClient.Get(host + "/v1/sources")
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

			var skipTLSVerify bool

			_, err = res.TLS.PeerCertificates[0].Verify(opts)
			if err != nil {
				if _, ok := err.(x509.UnknownAuthorityError); !ok {
					return err
				}

				proceed := false
				fmt.Println()
				fmt.Print("Could not verify certificate for host ")
				fmt.Print(termenv.String(args[0]).Bold())
				fmt.Println()
				prompt := &survey.Confirm{
					Message: "Are you sure you want to continue (yes/no)?",
				}

				p := termenv.ColorProfile()

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

				skipTLSVerify = true
			}

			var response struct{ Data []registry.Source }
			sourcesErr := checkAndDecode(res, &response)

			// TODO (jmorganca): make this check more reliable - i.e. a fixed error message
			// or an alternative api endpoint to check if infra is "locked"
			if sourcesErr != nil && sourcesErr.Error() != "no admin user" {
				return err
			}

			httpClient, err := client(host, "", skipTLSVerify)
			if err != nil {
				return err
			}

			var loginOrSignupRes *http.Response

			// Log in or prompt admin signup if no admins exist
			if sourcesErr != nil && sourcesErr.Error() == "no admin user" {
				fmt.Println()
				fmt.Println(blue("Welcome to Infra. Get started by creating your admin user:"))
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

				form := url.Values{}
				form.Add("email", email)
				form.Add("password", password)

				req, err := http.NewRequest("POST", host+"/v1/signup", strings.NewReader(form.Encode()))
				if err != nil {
					return err
				}

				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				fmt.Println(blue("✓") + " Creating admin user...")
				loginOrSignupRes, err = httpClient.Do(req)
				if err != nil {
					return err
				}
			} else {
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
						Message: "Choose a login method:",
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

				loginOrSignupRes, err = httpClient.Do(req)
				if err != nil {
					return err
				}
			}

			var loginOrSignupResponse struct {
				Token string `json:"token"`
			}
			err = checkAndDecode(loginOrSignupRes, &loginOrSignupResponse)
			if err != nil {
				return err
			}

			logout()

			config := &Config{
				Host:          host,
				Token:         loginOrSignupResponse.Token,
				SkipTLSVerify: true,
			}

			err = writeConfig(config)
			if err != nil {
				return err
			}

			fmt.Println(blue("✓") + " Logged in")

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

			updated, err := updateKubeconfig()
			if err != nil {
				return err
			}

			if updated {
				fmt.Println(blue("✓") + " Kubeconfig updated")
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolP("skip-tls-verify", "k", false, "skip TLS verification")

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

	registryUrl, err := registryUrl(config.Host)
	if err != nil {
		return err
	}

	if config.Token == "" {
		return nil
	}

	_, err = httpClient.Post(registryUrl.String()+"/v1/logout", "application/x-www-form-urlencoded", nil)
	if err != nil {
		return err
	}

	err = removeConfig()
	if err != nil {
		return err
	}

	// Kill client
	home, err := homedir.Dir()
	if err != nil {
		return err
	}

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
	os.Remove(filepath.Join(home, ".infra", "destinations"))

	return nil
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of an Infra Registry",
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

var destinationCmd = &cobra.Command{
	Use:   "destination",
	Short: "Manage infrastructure destinations",
}

var destinationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List destinations",
	RunE: func(cmd *cobra.Command, args []string) error {
		destinations, err := fetchDestinations()
		if err != nil {
			return err
		}

		sort.Slice(destinations, func(i, j int) bool {
			return destinations[i].Created > destinations[j].Created
		})

		kubeConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()

		rows := [][]string{}
		for _, dest := range destinations {
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

		_, err = updateKubeconfig()
		if err != nil {
			return err
		}

		return nil
	},
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userCreateCmd = &cobra.Command{
	Use:     "create EMAIL PASSWORD",
	Short:   "create a user",
	Args:    cobra.ExactArgs(2),
	Example: "$ infra user create admin@example.com password",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		registryUrl, err := registryUrl(config.Host)
		if err != nil {
			return err
		}

		email := args[0]
		password := args[1]
		form := url.Values{}
		form.Add("email", email)
		form.Add("password", password)

		res, err := httpClient.PostForm(registryUrl.String()+"/v1/users", form)
		if err != nil {
			return err
		}

		var user registry.User
		err = checkAndDecode(res, &user)
		if err != nil {
			return err
		}

		return nil
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete USER",
	Short: "delete a user",
	Args:  cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra user delete user@example.com`),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		registryUrl, err := registryUrl(config.Host)
		if err != nil {
			return err
		}

		params := url.Values{}
		params.Set("email", args[0])

		res, err := httpClient.Get(registryUrl.String() + "/v1/users?" + params.Encode())
		if err != nil {
			return err
		}

		var listResponse struct {
			Data []registry.User `json:"data"`
		}
		err = checkAndDecode(res, &listResponse)
		if err != nil {
			return err
		}

		for _, u := range listResponse.Data {
			req, err := http.NewRequest(http.MethodDelete, registryUrl.String()+"/v1/users/"+u.ID, nil)
			if err != nil {
				log.Fatal(err)
			}

			res, err := httpClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			var response registry.DeleteResponse
			err = checkAndDecode(res, &response)
			if err != nil {
				return err
			}

			res.Body.Close()
		}

		return nil
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		registryUrl, err := registryUrl(config.Host)
		if err != nil {
			return err
		}

		res, err := httpClient.Get(registryUrl.String() + "/v1/users")
		if err != nil {
			return err
		}

		var response struct {
			Data []registry.User `json:"data"`
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
			sources := ""
			for i, s := range user.Sources {
				if i > 0 {
					sources += ","
				}
				sources += s.Kind
			}

			admin := ""
			if user.Admin {
				admin = "x"
			}

			rows = append(rows, []string{user.Email, sources, units.HumanDuration(time.Now().UTC().Sub(time.Unix(user.Created, 0))) + " ago", admin})
		}

		printTable([]string{"EMAIL", "SOURCE", "CREATED", "ADMIN"}, rows)

		return nil
	},
}

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage identity sources",
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		registryUrl, err := registryUrl(config.Host)
		if err != nil {
			return err
		}

		res, err := httpClient.Get(registryUrl.String() + "/v1/sources")
		if err != nil {
			return err
		}

		var response struct{ Data []registry.Source }
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
			rows = append(rows, []string{source.ID, source.Kind, units.HumanDuration(time.Now().UTC().Sub(time.Unix(source.Created, 0))) + " ago", info})
		}

		printTable([]string{"SOURCE ID", "KIND", "CREATED", "DESCRIPTION"}, rows)

		return nil
	},
}

func newSourceCreateCmd() *cobra.Command {
	var apiToken, domain, clientID, clientSecret string

	cmd := &cobra.Command{
		Use:   "create KIND",
		Short: "Connect an identity source",
		Args:  cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			$ infra source create okta \
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

			registryUrl, err := registryUrl(config.Host)
			if err != nil {
				return err
			}

			form := url.Values{}
			form.Add("kind", args[0])
			form.Add("apiToken", apiToken)
			form.Add("domain", domain)
			form.Add("clientID", clientID)
			form.Add("clientSecret", clientSecret)

			res, err := httpClient.PostForm(registryUrl.String()+"/v1/sources", form)
			if err != nil {
				return err
			}

			var source registry.Source
			err = checkAndDecode(res, &source)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&apiToken, "api-token", "", "Api Token")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain (e.g. example.okta.com)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID for single sign on")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Client Secret for single sign on")

	return cmd
}

var sourceDeleteCmd = &cobra.Command{
	Use:     "delete ID",
	Aliases: []string{"rm"},
	Short:   "Delete an identity source",
	Args:    cobra.ExactArgs(1),
	Example: heredoc.Doc(`
			$ infra source delete n7bha2pxjpa01a`),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		registryUrl, err := registryUrl(config.Host)
		if err != nil {
			return err
		}

		id := args[0]
		req, err := http.NewRequest(http.MethodDelete, registryUrl.String()+"/v1/sources/"+id, nil)
		if err != nil {
			log.Fatal(err)
		}

		res, err := httpClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		var response registry.DeleteResponse
		err = checkAndDecode(res, &response)
		if err != nil {
			return err
		}

		return nil
	},
}

var apikeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "Manage API Keys",
}

var apikeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API Keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		httpClient, err := client(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		registryUrl, err := registryUrl(config.Host)
		if err != nil {
			return err
		}

		res, err := httpClient.Get(registryUrl.String() + "/v1/apikeys")
		if err != nil {
			return err
		}

		var response struct{ Data []registry.APIKey }
		err = checkAndDecode(res, &response)
		if err != nil {
			return err
		}

		sort.Slice(response.Data, func(i, j int) bool {
			return response.Data[i].Created > response.Data[j].Created
		})

		rows := [][]string{}
		for _, apikey := range response.Data {
			rows = append(rows, []string{apikey.Name, apikey.Key})
		}

		printTable([]string{"NAME", "APIKEY"}, rows)

		return nil
	},
}

func newRegistryCmd() (*cobra.Command, error) {
	var options registry.Options

	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Start Infra Registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return registry.Run(options)
		},
	}

	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	cmd.Flags().StringVarP(&options.ConfigPath, "config", "c", "", "config file")
	cmd.Flags().StringVar(&options.DBPath, "db", filepath.Join(home, ".infra", "infra.db"), "path to database file")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(home, ".infra", "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")

	return cmd, nil
}

func newEngineCmd() *cobra.Command {
	var options engine.Options

	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Start Infra Engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.Registry == "" {
				return errors.New("registry not specified (--registry or INFRA_ENGINE_REGISTRY)")
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

	skipTLSVerify := true
	// TODO (jmorganca): warn users instead of skipping TLS verification
	// OR find a way to include the server certificate in the api key
	// skipTLSVerify := len(os.Getenv("INFRA_ENGINE_SKIP_TLS_VERIFY")) > 0
	cmd.PersistentFlags().BoolVarP(&options.SkipTLSVerify, "skip-tls-verify", "k", skipTLSVerify, "skip TLS verification")
	cmd.Flags().StringVarP(&options.Registry, "registry", "r", os.Getenv("INFRA_ENGINE_REGISTRY"), "registry hostname")
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

		registryUrl, err := registryUrl(config.Host)
		if err != nil {
			return err
		}

		if config.Token == "" {
			return nil
		}

		res, err := httpClient.Post(registryUrl.String()+"/v1/creds", "application/x-www-form-urlencoded", nil)
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

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(logoutCmd)

	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userDeleteCmd)
	rootCmd.AddCommand(userCmd)

	destinationCmd.AddCommand(destinationListCmd)
	rootCmd.AddCommand(destinationCmd)

	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(newSourceCreateCmd())
	sourceCmd.AddCommand(sourceDeleteCmd)
	rootCmd.AddCommand(sourceCmd)

	apikeyCmd.AddCommand(apikeyListCmd)
	rootCmd.AddCommand(apikeyCmd)

	registryCmd, err := newRegistryCmd()
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(registryCmd)
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
