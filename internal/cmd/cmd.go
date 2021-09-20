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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1alpha1 "k8s.io/client-go/pkg/apis/clientauthentication/v1alpha1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Config struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
	SourceID      string `json:"source-id"`
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

func NewApiContext(token string) context.Context {
	return context.WithValue(context.Background(), api.ContextAccessToken, token)
}

func NewApiClient(host string, skipTLSVerify bool) (*api.APIClient, error) {
	u, err := urlx.Parse(host)
	if err != nil {
		return nil, err
	}

	config := api.NewConfiguration()
	config.Host = u.Host
	config.Scheme = "https"
	if skipTLSVerify {
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return api.NewAPIClient(config), nil
}

func apiContextFromConfig() (context.Context, error) {
	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	return NewApiContext(config.Token), nil
}

func apiClientFromConfig() (*api.APIClient, error) {
	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	return NewApiClient(config.Host, config.SkipTLSVerify)
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
				Args:       []string{"creds", d.Name},
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

func promptShouldSkipTLSVerify(host string, skipTLSVerify bool) (shouldSkipTLSVerify bool, proceed bool, err error) {
	if skipTLSVerify {
		return true, true, nil
	}

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
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, "Could not verify certificate for host ")
		fmt.Fprint(os.Stderr, termenv.String(host).Bold())
		fmt.Fprintln(os.Stderr)
		prompt := &survey.Confirm{
			Message: "Are you sure you want to continue?",
		}

		err := survey.AskOne(prompt, &proceed, survey.WithIcons(func(icons *survey.IconSet) {
			icons.Question.Text = blue("?")
		}))
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return false, false, err
		}

		if !proceed {
			return false, false, nil
		}

		return true, true, nil
	}

	return false, true, nil
}

func promptSelectSource(sources []api.Source, sourceID string) (*api.Source, error) {
	if len(sourceID) > 0 {
		for _, source := range sources {
			if source.Id == sourceID {
				return &source, nil
			}
		}
		return nil, errors.New("source not found")
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Created > sources[j].Created
	})

	options := []string{}
	for _, s := range sources {
		switch {
		case s.Okta != nil:
			options = append(options, fmt.Sprintf("Okta [%s]", s.Okta.Domain))
		}
	}

	var option int
	prompt := &survey.Select{
		Message: "Choose a login method:",
		Options: options,
	}
	err := survey.AskOne(prompt, &option, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = blue("?")
	}))
	if err == terminal.InterruptErr {
		return nil, err
	}

	return &sources[option], nil
}

func login(config *Config) error {
	skipTLSVerify, proceed, err := promptShouldSkipTLSVerify(config.Host, config.SkipTLSVerify)
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	client, err := NewApiClient(config.Host, skipTLSVerify)
	if err != nil {
		return err
	}

	sources, _, err := client.SourcesApi.ListSources(context.Background()).Execute()
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		return errors.New("no sources configured")
	}

	source, err := promptSelectSource(sources, config.SourceID)
	switch err {
	case nil:
	case terminal.InterruptErr:
		return nil
	default:
		return err
	}

	var loginReq api.LoginRequest

	switch {
	case source.Okta != nil:
		// Start OIDC flow
		// Get auth code from Okta
		// Send auth code to Infra to login as a user
		state := generate.RandString(12)
		authorizeUrl := "https://" + source.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + source.Okta.ClientId + "&response_type=code&scope=openid+email&nonce=" + generate.RandString(10) + "&state=" + state

		fmt.Fprintln(os.Stderr, blue("✓")+" Logging in with Okta...")
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
		return errors.New("invalid source selected")
	}

	loginRes, _, err := client.AuthApi.Login(context.Background()).Body(loginReq).Execute()
	if err != nil {
		return err
	}

	err = writeConfig(&Config{
		Name:          loginRes.Name,
		Token:         loginRes.Token,
		Host:          config.Host,
		SkipTLSVerify: skipTLSVerify,
		SourceID:      source.Id,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, blue("✓")+" Logged in as "+termenv.String(loginRes.Name).Bold().String())

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

	client, err = NewApiClient(config.Host, skipTLSVerify)
	if err != nil {
		return err
	}
	ctx := NewApiContext(loginRes.Token)

	destinations, _, err := client.DestinationsApi.ListDestinations(ctx).Execute()
	if err != nil {
		return err
	}

	err = updateKubeconfig(destinations)
	if err != nil {
		return err
	}

	if len(destinations) > 0 {
		kubeConfigPath := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ConfigAccess().GetDefaultFilename()
		fmt.Fprintln(os.Stderr, blue("✓")+" Kubeconfig updated: "+termenv.String(strings.ReplaceAll(kubeConfigPath, home, "~")).Bold().String())
	}

	context, err := switchToFirstInfraContext()
	if err != nil {
		return err
	}

	if context != "" {
		fmt.Fprintln(os.Stderr, blue("✓")+" Kubernetes current context is now "+termenv.String(context).Bold().String())
	}

	return nil
}

var loginCmd = &cobra.Command{
	Use:     "login REGISTRY",
	Short:   "Login to an Infra Registry",
	Args:    cobra.ExactArgs(1),
	Example: "$ infra login infra.example.com",
	RunE: func(cmd *cobra.Command, args []string) error {
		return login(&Config{Host: args[0]})
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

		client, err := NewApiClient(config.Host, config.SkipTLSVerify)
		if err != nil {
			return err
		}

		_, err = client.AuthApi.Logout(NewApiContext(config.Token)).Execute()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

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
		client, err := apiClientFromConfig()
		if err != nil {
			return err
		}

		ctx, err := apiContextFromConfig()
		if err != nil {
			return err
		}

		destinations, _, err := client.DestinationsApi.ListDestinations(ctx).Execute()
		if err != nil {
			return err
		}

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
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "To connect, run \"kubectl config use-context <name>\"")
		fmt.Fprintln(os.Stderr)

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
		config, err := readConfig()
		if err != nil {
			return err
		}

		client, err := apiClientFromConfig()
		if err != nil {
			return err
		}

		ctx, err := apiContextFromConfig()
		if err != nil {
			return err
		}

		users, _, err := client.UsersApi.ListUsers(ctx).Execute()
		if err != nil {
			return err
		}

		sort.Slice(users, func(i, j int) bool {
			return users[i].Created > users[j].Created
		})

		rows := [][]string{}
		for _, u := range users {
			email := u.Email
			if email == config.Name {
				email += "*"
			}
			rows = append(rows, []string{email, units.HumanDuration(time.Now().UTC().Sub(time.Unix(u.Created, 0))) + " ago"})
		}

		printTable([]string{"EMAIL", "CREATED"}, rows)

		return nil
	},
}

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClientFromConfig()
		if err != nil {
			return err
		}

		ctx, err := apiContextFromConfig()
		if err != nil {
			return err
		}

		groups, _, err := client.GroupsApi.ListGroups(ctx).Execute()
		if err != nil {
			return err
		}

		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Created > groups[j].Created
		})

		rows := [][]string{}
		for _, g := range groups {
			rows = append(rows, []string{g.Name, units.HumanDuration(time.Now().UTC().Sub(time.Unix(g.Created, 0))) + " ago", g.Source})
		}

		printTable([]string{"NAME", "CREATED", "SOURCE"}, rows)
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
	cmd.Flags().BoolVar(&options.UI, "ui", false, "enable ui")
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

		client, err := apiClientFromConfig()
		if err != nil {
			fmt.Fprintln(w, blue("✕")+" Could not retrieve client version")
			return err
		}

		// Note that we use the client to get this version, but it is in fact the server version
		res, _, err := client.VersionApi.Version(context.Background()).Execute()
		if err != nil {
			fmt.Fprintln(w, "Registry:\t", "not connected")
			return err
		}

		fmt.Fprintln(w, "Registry:\t", res.Version)
		fmt.Fprintln(w)

		return nil
	},
}

var credsCmd = &cobra.Command{
	Use:   "creds",
	Short: "Generate a JWT token for connecting to a destination, eg k8s",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClientFromConfig()
		if err != nil {
			return err
		}

		ctx, err := apiContextFromConfig()
		if err != nil {
			return err
		}

		if len(args) != 1 {
			return errors.New("expecting destination as argument")
		}

		destination := args[0]
		execCredential := &clientauthenticationv1alpha1.ExecCredential{}

		err = getCache("dest_tokens", destination, execCredential)
		if err != nil {
			return err
		}

		if isExpired(execCredential) {
			credReq := client.CredsApi.CreateCred(ctx).Body(api.CredRequest{Destination: &destination})
			cred, res, err := credReq.Execute()
			if err != nil {
				switch res.StatusCode {
				case http.StatusForbidden:
					config, err := readConfig()
					if err != nil {
						return err
					}

					err = login(config)
					if err != nil {
						return err
					}

					ctx, err := apiContextFromConfig()
					if err != nil {
						return err
					}

					cred, _, err = client.CredsApi.CreateCred(ctx).Execute()
					if err != nil {
						return err
					}

				default:
					return err
				}
			}

			execCredential = &clientauthenticationv1alpha1.ExecCredential{
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
			setCache("dest_tokens", destination, execCredential)
		}

		bts, err := json.Marshal(execCredential)
		if err != nil {
			return err
		}

		startProxy()

		fmt.Println(string(bts))

		return nil
	},
}

var clientCmd = &cobra.Command{
	Use:    "client",
	Short:  "Run local client to relay requests",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "Starting client")
		return RunLocalClient()
	},
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(groupsCmd)
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

// getCache populates obj with whatever is in the cache
func getCache(path, name string, obj interface{}) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path = filepath.Join(home, ".infra", "cache", path)
	fullPath := filepath.Join(path, name)
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	f, err := os.Open(fullPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(obj); err != nil {
		return err
	}

	return nil
}

func setCache(path, name string, obj interface{}) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path = filepath.Join(home, ".infra", "cache", path)
	fullPath := filepath.Join(path, name)

	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	if err := e.Encode(obj); err != nil {
		return err
	}

	return nil
}

// isExpired returns true if the credential is expired or incomplete.
// it only returns false if the credential is good to use.
func isExpired(cred *clientauthenticationv1alpha1.ExecCredential) bool {
	if cred == nil {
		return true
	}
	if cred.Status == nil {
		return true
	}
	if cred.Status.ExpirationTimestamp == nil {
		return true
	}
	// make sure it expires in more than 1 second from now.
	now := time.Now().Add(1 * time.Second)
	// only valid if it hasn't expired yet
	return cred.Status.ExpirationTimestamp.Time.Before(now)
}

func startProxy() error {
	home, err := os.UserHomeDir()
	if err != nil {
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

		tick := time.NewTicker(25 * time.Millisecond)
		timeout := time.NewTimer(10 * time.Second)
	Loop:
		for {
			select {
			case <-tick.C:
				_, err = os.Stat(filepath.Join(home, ".infra", "client", "pid"))
				if err == nil {
					break Loop
				}
			case <-timeout.C:
				return errors.New("timeout waiting for local client to start")
			}
		}
	}

	return nil
}
