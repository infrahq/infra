package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/docker/go-units"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry"
	"github.com/infrahq/infra/internal/version"
	"github.com/mitchellh/go-homedir"
	"github.com/muesli/termenv"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type ErrUnauthenticated struct{}

func (e *ErrUnauthenticated) Error() string {
	return "could not read local credentials. Are you logged in? To login, use \"infra login\""
}

func readConfig() (config *Config, err error) {
	config = &Config{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "config"))
	if os.IsNotExist(err) {
		return nil, &ErrUnauthenticated{}
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

func NewClient(host string, token string, skipTlsVerify bool) (*api.ClientWithResponses, error) {
	if host == "" {
		return nil, errors.New("host must not be empty")
	}

	u, err := urlx.Parse(host)
	if err != nil {
		return nil, err
	}

	u.Scheme = "https"

	bearerTokenProvider, err := securityprovider.NewSecurityProviderBearerToken(token)
	if err != nil {
		return nil, err
	}

	client, err := api.NewClient(u.String()+"/v1", api.WithRequestEditorFn(bearerTokenProvider.Intercept))
	if err != nil {
		return nil, err
	}

	if skipTlsVerify {
		client.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return &api.ClientWithResponses{ClientInterface: client}, nil
}

func clientFromConfig() (*api.ClientWithResponses, error) {
	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	return NewClient(config.Host, config.Token, config.SkipTLSVerify)
}

func clientConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.WarnIfAllMissing = false
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
}

func updateKubeconfig(destinations []api.Destination) error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	if len(destinations) > 0 {
		destinationsJSON, err := json.Marshal(destinations)
		if err != nil {
			return err
		}

		// Write destinations to a known json file location for `infra client` to read
		err = os.WriteFile(filepath.Join(home, ".infra", "destinations"), destinationsJSON, 0644)
		if err != nil {
			return err
		}
	}

	defaultConfig := clientConfig()
	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return err
	}

	for _, d := range destinations {
		contextName := "infra:" + d.Name

		kubeConfig.Clusters[contextName] = &clientcmdapi.Cluster{
			Server:               "https://localhost:32710/client/" + d.Name,
			CertificateAuthority: filepath.Join(home, ".infra", "client", "cert.pem"),
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[contextName] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:    executable,
				Args:       []string{"creds"},
				APIVersion: "client.authentication.k8s.io/v1alpha1",
			},
		}

		kubeConfig.Contexts[contextName] = &clientcmdapi.Context{
			Cluster:  contextName,
			AuthInfo: contextName,
		}
	}

	for name := range kubeConfig.Contexts {
		if !strings.HasPrefix(name, "infra:") {
			continue
		}

		destinationName := strings.ReplaceAll(name, "infra:", "")

		var exists bool
		for _, d := range destinations {
			if destinationName == d.Name {
				exists = true
			}
		}

		if !exists {
			delete(kubeConfig.Clusters, name)
			delete(kubeConfig.Contexts, name)
			delete(kubeConfig.AuthInfos, name)
		}
	}

	if len(destinations) == 0 {
		_, ok := kubeConfig.Contexts[kubeConfig.CurrentContext]
		if !ok {
			kubeConfig.CurrentContext = ""
			for name := range kubeConfig.Contexts {
				kubeConfig.CurrentContext = name
				break
			}
		}
	}

	if err = clientcmd.WriteToFile(kubeConfig, defaultConfig.ConfigAccess().GetDefaultFilename()); err != nil {
		return err
	}

	return nil
}

func switchToFirstInfraContext() (string, error) {
	defaultConfig := clientConfig()
	kubeConfig, err := defaultConfig.RawConfig()
	if err != nil {
		return "", err
	}

	resultContext := ""

	if kubeConfig.Contexts[kubeConfig.CurrentContext] != nil && strings.HasPrefix(kubeConfig.CurrentContext, "infra:") {
		// if the current context is an infra-controlled context, stay there
		resultContext = kubeConfig.CurrentContext
	} else {
		for c := range kubeConfig.Contexts {
			if strings.HasPrefix(c, "infra:") {
				resultContext = c
				break
			}
		}
	}

	if resultContext != "" {
		kubeConfig.CurrentContext = resultContext
		if err = clientcmd.WriteToFile(kubeConfig, defaultConfig.ConfigAccess().GetDefaultFilename()); err != nil {
			return "", err
		}
	}

	return resultContext, nil
}

var rootCmd = &cobra.Command{
	Use:   "infra",
	Short: "Infrastructure Identity & Access Management (IAM)",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

func promptShouldSkipTLSVerify(host string) (skipTlsVerify bool, proceed bool, err error) {
	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	url, err := urlx.Parse(host)
	if err != nil {
		return false, false, err
	}
	url.Scheme = "https"
	urlString := url.String()

	_, err = httpClient.Get(urlString)
	if err != nil {
		if !errors.As(err, &x509.UnknownAuthorityError{}) && !errors.As(err, &x509.HostnameError{}) {
			return false, false, err
		}

		proceed := false
		fmt.Println()
		fmt.Print("Could not verify certificate for host ")
		fmt.Print(termenv.String(host).Bold())
		fmt.Println()
		prompt := &survey.Confirm{
			Message: "Are you sure you want to continue?",
		}

		err := survey.AskOne(prompt, &proceed, survey.WithIcons(func(icons *survey.IconSet) {
			icons.Question.Text = blue("?")
		}))
		if err != nil {
			fmt.Println(err.Error())
			return false, false, err
		}

		if !proceed {
			return false, false, nil
		}

		return true, true, nil
	}

	return false, true, nil
}

var loginCmd = &cobra.Command{
	Use:     "login REGISTRY",
	Short:   "Login to an Infra Registry",
	Args:    cobra.ExactArgs(1),
	Example: "$ infra login infra.example.com",
	RunE: func(cmd *cobra.Command, args []string) error {
		skipTLSVerify, proceed, err := promptShouldSkipTLSVerify(args[0])
		if err != nil {
			return err
		}

		if !proceed {
			return nil
		}

		client, err := NewClient(args[0], "", skipTLSVerify)
		if err != nil {
			return err
		}

		res, err := client.StatusWithResponse(context.Background())
		if err != nil {
			return err
		}

		var authRes *api.AuthResponse
		if !res.JSON200.Admin {
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

			fmt.Println(blue("✓") + " Creating admin user...")
			res, err := client.SignupWithResponse(context.Background(), api.SignupJSONRequestBody{Email: email, Password: password})
			if err != nil {
				return err
			}

			authRes = res.JSON200
		} else {
			sourcesRes, err := client.ListSourcesWithResponse(context.Background())
			if err != nil {
				return err
			}

			sources := *sourcesRes.JSON200
			sort.Slice(sources, func(i, j int) bool {
				return sources[i].Created > sources[j].Created
			})

			options := []string{}
			for _, s := range sources {
				switch {
				case s.Okta != nil:
					options = append(options, fmt.Sprintf("Okta [%s]", s.Okta.Domain))
				default:
					options = append(options, "Username & password")
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

			source := sources[option]
			var loginReq api.LoginJSONRequestBody

			switch {
			case source.Okta != nil:
				// Start OIDC flow
				// Get auth code from Okta
				// Send auth code to Infra to login as a user
				state := generate.RandString(12)
				authorizeUrl := "https://" + source.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + source.Okta.ClientId + "&response_type=code&scope=openid+email&nonce=" + generate.RandString(10) + "&state=" + state

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

				loginReq.Okta = &api.LoginRequestOkta{
					Domain: source.Okta.Domain,
					Code:   code,
				}
			default:
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

				fmt.Println(blue("✓") + " Logging in with username & password...")

				loginReq.Infra = &api.LoginRequestInfra{
					Email:    email,
					Password: password,
				}
			}

			loginRes, err := client.LoginWithResponse(context.Background(), loginReq)
			if err != nil {
				return err
			}

			authRes = loginRes.JSON200
		}

		config := &Config{
			Host:          args[0],
			Token:         authRes.Token,
			SkipTLSVerify: skipTLSVerify,
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
			certBytes, keyBytes, err := generate.SelfSignedCert([]string{"localhost", "localhost:32710"})
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

		loggedInClient, err := NewClient(args[0], authRes.Token, skipTLSVerify)
		if err != nil {
			return err
		}

		destRes, err := loggedInClient.ListDestinationsWithResponse(context.Background())
		if err != nil {
			return err
		}

		destinations := *destRes.JSON200

		err = updateKubeconfig(destinations)
		if err != nil {
			return err
		}

		if len(destinations) > 0 {
			kubeConfigPath := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ConfigAccess().GetDefaultFilename()
			fmt.Println(blue("✓") + " Kubeconfig updated: " + termenv.String(strings.ReplaceAll(kubeConfigPath, home, "~")).Bold().String())
		}

		context, err := switchToFirstInfraContext()
		if err != nil {
			return err
		}

		if context != "" {
			fmt.Println(blue("✓") + " Kubernetes current context is now " + termenv.String(context).Bold().String())
		}

		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout of an Infra Registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		if config.Token == "" {
			return nil
		}

		client, err := NewClient(config.Host, config.Token, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		client.Logout(context.Background())

		err = removeConfig()
		if err != nil {
			return err
		}

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

		return updateKubeconfig([]api.Destination{})
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List clusters",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.ListDestinationsWithResponse(context.Background())
		if err != nil {
			return err
		}

		destinations := *res.JSON200

		sort.Slice(destinations, func(i, j int) bool {
			return destinations[i].Created > destinations[j].Created
		})

		kubeConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).RawConfig()

		rows := [][]string{}
		for _, d := range destinations {
			switch {
			case d.Kubernetes != nil:
				star := ""
				if d.Name == kubeConfig.CurrentContext {
					star = "*"
				}
				rows = append(rows, []string{"infra:" + d.Name + star, d.Kubernetes.Endpoint})
			}
		}

		printTable([]string{"NAME", "ENDPOINT"}, rows)
		fmt.Println()
		fmt.Println("To connect, run \"kubectl config use-context <name>\"")
		fmt.Println()

		err = updateKubeconfig(destinations)
		if err != nil {
			return err
		}

		return nil
	},
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List users",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.ListUsersWithResponse(context.Background())
		if err != nil {
			return err
		}

		users := *res.JSON200

		sort.Slice(users, func(i, j int) bool {
			return users[i].Created > users[j].Created
		})

		rows := [][]string{}
		for _, u := range users {
			admin := ""
			if u.Admin {
				admin = "x"
			}

			rows = append(rows, []string{u.Email, units.HumanDuration(time.Now().UTC().Sub(time.Unix(u.Created, 0))) + " ago", admin})
		}

		printTable([]string{"EMAIL", "CREATED", "ADMIN"}, rows)

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
	cmd.Flags().StringVar(&options.DefaultApiKey, "initial-apikey", os.Getenv("INFRA_REGISTRY_DEFAULT_API_KEY"), "initial api key for adding destinations")
	cmd.Flags().StringVar(&options.DBPath, "db", filepath.Join(home, ".infra", "infra.db"), "path to database file")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(home, ".infra", "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")
	cmd.Flags().StringVar(&options.UIProxy, "ui-proxy", "", "proxy ui requests to this host")

	defaultSync := 30
	osSync := os.Getenv("INFRA_SYNC_INTERVAL_SECONDS")
	if osSync != "" {
		defaultSync, err = strconv.Atoi(osSync)
		if err != nil {
			logging.L.Error("could not convert INFRA_SYNC_INTERVAL_SECONDS to an integer: " + err.Error())
		}
	}
	cmd.Flags().IntVar(&options.SyncInterval, "sync-interval", defaultSync, "the interval (in seconds) at which Infra will poll sources for users and groups")

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
			if options.Registry != "infra" && options.APIKey == "" {
				return errors.New("api-key not specified (--api-key or INFRA_ENGINE_API_KEY)")
			}
			return engine.Run(options)
		},
	}

	cmd.PersistentFlags().BoolVar(&options.ForceTLSVerify, "force-tls-verify", false, "force TLS verification")
	cmd.Flags().StringVarP(&options.Registry, "registry", "r", os.Getenv("INFRA_ENGINE_REGISTRY"), "registry hostname")
	cmd.Flags().StringVarP(&options.Name, "name", "n", os.Getenv("INFRA_ENGINE_NAME"), "cluster name")
	cmd.Flags().StringVarP(&options.Endpoint, "endpoint", "e", os.Getenv("INFRA_ENGINE_ENDPOINT"), "cluster endpoint")
	cmd.Flags().StringVar(&options.APIKey, "api-key", os.Getenv("INFRA_ENGINE_API_KEY"), "api key")

	return cmd
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Display the Infra build version",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
		defer w.Flush()
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Client:\t", version.Version)

		client, err := clientFromConfig()
		if err != nil {
			fmt.Fprintln(w, blue("✕")+" Could not retrieve client version")
			return err
		}

		// Note that we use the client to get this version, but it is in fact the server version
		res, err := client.VersionWithResponse(context.Background())
		if err != nil {
			status, ok := status.FromError(err)
			if !ok {
				return err
			}
			switch status.Code() {
			case codes.Unavailable:
				fmt.Fprintln(w, "Registry:\t", "not connected")
				return nil
			default:
				return err
			}
		}

		fmt.Fprintln(w, "Registry:\t", res.JSON200.Version)
		fmt.Fprintln(w)

		return nil
	},
}

var credsCmd = &cobra.Command{
	Use:    "creds",
	Short:  "Generate credentials",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := clientFromConfig()
		if err != nil {
			return err
		}

		res, err := client.CreateCredWithResponse(context.Background())
		if err != nil {
			return err
		}

		cred := *res.JSON200

		execCredential := &clientauthenticationv1alpha1.ExecCredential{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ExecCredential",
				APIVersion: clientauthenticationv1alpha1.SchemeGroupVersion.String(),
			},
			Spec: clientauthenticationv1alpha1.ExecCredentialSpec{},
			Status: &clientauthenticationv1alpha1.ExecCredentialStatus{
				Token:               cred.Token,
				ExpirationTimestamp: &metav1.Time{Time: time.Unix(cred.Expires, 0)},
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
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)

	registryCmd, err := newRegistryCmd()
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(registryCmd)
	rootCmd.AddCommand(newEngineCmd())

	rootCmd.AddCommand(versionCmd)

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
