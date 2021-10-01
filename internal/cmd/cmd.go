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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/gofrs/flock"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/engine"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry"
	"github.com/lensesio/tableprinter"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	clientauthenticationv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
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
	return "Could not read local credentials. Are you logged in? Use \"infra login\" to login."
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

	if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "config"), []byte(contents), 0o600); err != nil {
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

func printTable(data interface{}) {
	table := tableprinter.New(os.Stdout)

	table.AutoFormatHeaders = true
	table.HeaderAlignment = tableprinter.AlignLeft
	table.AutoWrapText = false
	table.DefaultAlignment = tableprinter.AlignLeft
	table.CenterSeparator = ""
	table.ColumnSeparator = ""
	table.RowSeparator = ""
	table.HeaderLine = false
	table.BorderBottom = false
	table.BorderLeft = false
	table.BorderRight = false
	table.BorderTop = false
	table.Print(data)
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
					//nolint:gosec // We may purposely set insecureskipverify via a flag
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if len(destinations) > 0 {
		destinationsJSON, err := json.Marshal(destinations)
		if err != nil {
			return err
		}

		// Write destinations to a known json file location for `infra client` to read
		err = os.WriteFile(filepath.Join(homeDir, ".infra", "destinations"), destinationsJSON, 0o600)
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
			Server:                   fmt.Sprintf("https://%s/proxy", d.Kubernetes.Endpoint),
			CertificateAuthorityData: []byte(d.Kubernetes.Ca),
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		kubeConfig.AuthInfos[contextName] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:    executable,
				Args:       []string{"token", d.Name},
				APIVersion: "client.authentication.k8s.io/v1beta1",
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

	url, err := urlx.Parse(host)
	if err != nil {
		return false, false, err
	}

	url.Scheme = "https"
	urlString := url.String()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, urlString, nil)
	if err != nil {
		return false, false, err
	}

	res, err := http.DefaultClient.Do(req)
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
	defer res.Body.Close()

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
		if s.Okta != nil {
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
	if errors.Is(err, terminal.InterruptErr) {
		return nil, err
	}

	return &sources[option], nil
}

func login(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	lock := flock.New(filepath.Join(homeDir, ".infra", "login.lock"))

	acquired, err := lock.TryLock()
	if err != nil {
		return err
	}

	defer func() {
		if err := lock.Unlock(); err != nil {
			fmt.Fprintln(os.Stderr, "failed to unlock login.lock")
		}
	}()

	if !acquired {
		fmt.Fprintln(os.Stderr, "another instance is trying to login")

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		_, err = lock.TryLockContext(ctx, time.Second*1)
		if err != nil {
			return err
		}

		return nil
	}

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

	switch {
	case err == nil:
	case errors.Is(err, terminal.InterruptErr):
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
		state, err := generate.RandString(12)
		if err != nil {
			return err
		}

		nonce, err := generate.RandString(10)
		if err != nil {
			return err
		}

		authorizeURL := "https://" + source.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + source.Okta.ClientId + "&response_type=code&scope=openid+email&nonce=" + nonce + "&state=" + state

		fmt.Fprintln(os.Stderr, blue("✓")+" Logging in with Okta...")

		ls, err := newLocalServer()
		if err != nil {
			return err
		}

		err = browser.OpenURL(authorizeURL)
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

	client, err = NewApiClient(config.Host, skipTLSVerify)
	if err != nil {
		return err
	}

	destinations, _, err := client.DestinationsApi.ListDestinations(NewApiContext(loginRes.Token)).Execute()
	if err != nil {
		return err
	}

	err = updateKubeconfig(destinations)
	if err != nil {
		return err
	}

	if len(destinations) > 0 {
		kubeConfigPath := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ConfigAccess().GetDefaultFilename()
		fmt.Fprintln(os.Stderr, blue("✓")+" Kubeconfig updated: "+termenv.String(strings.ReplaceAll(kubeConfigPath, homeDir, "~")).Bold().String())
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

func newLogoutCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout of an Infra Registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logout()
		},
	}

	return cmd, nil
}

func newListCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List destinations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list()
		},
	}

	return cmd, nil
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

	defaultInfraHome := filepath.Join("~", ".infra")

	cmd.Flags().StringVarP(&options.ConfigPath, "config", "c", "", "config file")
	cmd.Flags().StringVar(&options.RootAPIKey, "root-api-key", os.Getenv("INFRA_REGISTRY_ROOT_API_KEY"), "the root api key for privileged actions")
	cmd.Flags().StringVar(&options.EngineApiKey, "initial-engine-api-key", os.Getenv("ENGINE_API_KEY"), "initial api key for adding destinations")
	cmd.Flags().StringVar(&options.DBPath, "db", filepath.Join(defaultInfraHome, "infra.db"), "path to database file")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(defaultInfraHome, "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")
	cmd.Flags().BoolVar(&options.UI, "ui", false, "enable ui")
	cmd.Flags().StringVar(&options.UIProxy, "ui-proxy", "", "proxy ui requests to this host")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	if filepath.Dir(options.DBPath) == defaultInfraHome {
		options.DBPath = filepath.Join(homeDir, ".infra", "infra.db")
	}

	if filepath.Dir(options.TLSCache) == defaultInfraHome {
		options.TLSCache = filepath.Join(homeDir, ".infra", "cache")
	}

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

func newEngineCmd() (*cobra.Command, error) {
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

	defaultInfraHome := filepath.Join("~", ".infra")

	cmd.PersistentFlags().BoolVar(&options.ForceTLSVerify, "force-tls-verify", false, "force TLS verification")
	cmd.Flags().StringVarP(&options.Registry, "registry", "r", os.Getenv("INFRA_ENGINE_REGISTRY"), "registry hostname")
	cmd.Flags().StringVarP(&options.Name, "name", "n", os.Getenv("INFRA_ENGINE_NAME"), "cluster name")
	cmd.Flags().StringVar(&options.TLSCache, "tls-cache", filepath.Join(defaultInfraHome, "cache"), "path to directory to cache tls self-signed and Let's Encrypt certificates")
	cmd.Flags().StringVar(&options.APIKey, "api-key", os.Getenv("INFRA_ENGINE_API_KEY"), "api key")

	if filepath.Dir(options.TLSCache) == defaultInfraHome {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		options.TLSCache = filepath.Join(homeDir, ".infra", "cache")
	}

	return cmd, nil
}

func newVersionCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display the Infra build version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return version()
		},
	}

	return cmd, nil
}

func newTokenCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "token DESTINATION",
		Short: "Generate a JWT token for connecting to a destination, e.g. Kubernetes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("Expecting destination as an argument")
			}

			return token(args[0])
		},
	}

	return cmd, nil
}

func NewRootCmd() (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	logoutCmd, err := newLogoutCmd()
	if err != nil {
		return nil, err
	}

	listCmd, err := newListCmd()
	if err != nil {
		return nil, err
	}

	tokenCmd, err := newTokenCmd()
	if err != nil {
		return nil, err
	}

	versionCmd, err := newVersionCmd()
	if err != nil {
		return nil, err
	}

	registryCmd, err := newRegistryCmd()
	if err != nil {
		return nil, err
	}

	engineCmd, err := newEngineCmd()
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.AddCommand(registryCmd)
	rootCmd.AddCommand(engineCmd)

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path = filepath.Join(homeDir, ".infra", "cache", path)
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	fullPath := filepath.Join(path, name)

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path = filepath.Join(homeDir, ".infra", "cache", path)
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
func isExpired(cred *clientauthenticationv1beta1.ExecCredential) bool {
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
