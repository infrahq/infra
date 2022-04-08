package cmd

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/goware/urlx"
	"github.com/iancoleman/strcase"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
)

type loginCmdOptions struct {
	Server        string `mapstructure:"server"`
	AccessKey     string `mapstructure:"key"`
	Provider      string `mapstructure:"provider"`
	SkipTLSVerify bool   `mapstructure:"skipTLSVerify"`
}

type loginMethod int8

const (
	localLogin loginMethod = iota
	accessKeyLogin
	oidcLogin
)

const cliLoginRedirectURL = "http://localhost:8301"

func newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login [SERVER]",
		Short: "Login to Infra",
		Example: `
# By default, login will prompt for all required information.
$ infra login

# Login to a specified server
$ infra login SERVER
$ infra login --server SERVER

# Login with an access key
$ infra login --key KEY

# Login with a specified provider
$ infra login --provider NAME

# Use the '--non-interactive' flag to error out instead of prompting.
`,
		Args:  cobra.MaximumNArgs(1),
		Group: "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options loginCmdOptions
			strcase.ConfigureAcronym("skip-tls-verify", "skipTLSVerify")

			if err := parseOptions(cmd, &options, "INFRA"); err != nil {
				return err
			}

			if len(args) == 1 {
				if options.Server != "" {
					fmt.Fprintf(os.Stderr, "Server is specified twice. Ignoring flag [--server] and proceeding with %s", options.Server)
				}
				options.Server = args[0]
			}

			return login(options)
		},
	}

	cmd.Flags().String("key", "", "Login with an access key")
	cmd.Flags().String("server", "", "Infra server to login to")
	cmd.Flags().String("provider", "", "Login with an identity provider")
	cmd.Flags().Bool("skip-tls-verify", false, "Skip verifying server TLS certificates")
	return cmd
}

func login(options loginCmdOptions) error {
	var err error

	if options.Server == "" {
		options.Server, err = promptServer()
		if err != nil {
			return err
		}
	}

	client, err := newAPIClient(options.Server, options.SkipTLSVerify)
	if err != nil {
		return err
	}

	// If first-time setup needs to be run, accessKey is auto-populated
	setupRequired, err := client.SetupRequired()
	if err != nil {
		return err
	}
	if setupRequired.Required && options.AccessKey == "" {
		options.AccessKey, err = runSetupForLogin(client)
		if err != nil {
			return err
		}
	}

	loginReq := &api.LoginRequest{}

	switch {
	case options.AccessKey != "":
		loginReq.AccessKey = options.AccessKey
	case options.Provider != "":
		loginReq.OIDC, err = loginToProviderN(client, options.Provider)
		if err != nil {
			return err
		}
	default:
		loginMethod, provider, err := promptLoginOptions(client)
		if err != nil {
			return err
		}

		switch loginMethod {
		case accessKeyLogin:
			loginReq.AccessKey, err = promptAccessKeyLogin()
			if err != nil {
				return err
			}
		case localLogin:
			loginReq.PasswordCredentials, err = promptLocalLogin()
			if err != nil {
				return err
			}
		case oidcLogin:
			loginReq.OIDC, err = loginToProvider(provider)
			if err != nil {
				return err
			}
		}

	}

	return loginToInfra(client, loginReq)
}

func relogin() error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if isNonInteractiveMode() {
		return fmt.Errorf("Non-interactive login is not supported")
	}

	currentConfig, err := currentHostConfig()
	if err != nil {
		return err
	}

	client, err := apiClient(currentConfig.Host, "", currentConfig.SkipTLSVerify)
	if err != nil {
		return err
	}

	if currentConfig.ProviderID == 0 {
		return fmt.Errorf("Cannot renew login without provider")
	}

	provider, err := client.GetProvider(currentConfig.ProviderID)
	if err != nil {
		return err
	}

	code, err := oidcflow(provider.URL, provider.ClientID)
	if err != nil {
		return err
	}

	loginReq := &api.LoginRequest{
		OIDC: &api.LoginRequestOIDC{
			ProviderID:  provider.ID,
			RedirectURL: cliLoginRedirectURL,
			Code:        code,
		},
	}

	loginRes, err := client.Login(loginReq)
	if err != nil {
		return err
	}

	return finishLogin(currentConfig.Host, currentConfig.SkipTLSVerify, provider.ID, loginRes)
}

func loginToInfra(client *api.Client, loginReq *api.LoginRequest) error {
	loginRes, err := client.Login(loginReq)
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			switch {
			case loginReq.AccessKey != "":
				return &FailedLoginError{getLoggedInIdentityName(), accessKeyLogin}
			case loginReq.PasswordCredentials != nil:
				return &FailedLoginError{getLoggedInIdentityName(), localLogin}
			case loginReq.OIDC != nil:
				return &FailedLoginError{getLoggedInIdentityName(), oidcLogin}
			}
		}
		return err
	}

	fmt.Fprintf(os.Stderr, "  Logged in as %s\n", termenv.String(loginRes.Name).Bold().String())

	if err := updateInfraConfig(client, loginReq, loginRes); err != nil {
		return err
	}

	// Client needs to be refreshed from here onwards, based on the newly saved infra configuration.
	client, err = defaultAPIClient()
	if err != nil {
		return err
	}

	if err := updateKubeconfig(client, loginRes.PolymorphicID); err != nil {
		return err
	}

	if loginRes.PasswordUpdateRequired {
		if err := updateUserPassword(client, loginRes.PolymorphicID, loginReq.PasswordCredentials.Password); err != nil {
			return err
		}
	}

	return nil
}

// Updates all configs with the current logged in session
func updateInfraConfig(client *api.Client, loginReq *api.LoginRequest, loginRes *api.LoginResponse) error {
	clientHostConfig := ClientHostConfig{
		Current:       true,
		PolymorphicID: loginRes.PolymorphicID,
		Name:          loginRes.Name,
		AccessKey:     loginRes.AccessKey,
		Expires:       loginRes.Expires,
	}

	t, ok := client.HTTP.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("Could not update config due to an internal error")
	}
	clientHostConfig.SkipTLSVerify = t.TLSClientConfig.InsecureSkipVerify

	if loginReq.OIDC != nil {
		clientHostConfig.ProviderID = loginReq.OIDC.ProviderID
	}

	u, err := urlx.Parse(client.URL)
	if err != nil {
		return err
	}
	clientHostConfig.Host = u.Host

	if err := saveHostConfig(clientHostConfig); err != nil {
		return err
	}

	return nil
}

// TODO relogin(): Once relogin is revisited, delete finishLogin and use loginToInfra() instead
func finishLogin(host string, skipTLSVerify bool, providerID uid.ID, loginRes *api.LoginResponse) error {
	fmt.Fprintf(os.Stderr, "  Logged in as %s\n", termenv.String(loginRes.Name).Bold().String())

	config, err := readConfig()
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return err
	}

	if config == nil {
		config = NewClientConfig()
	}

	var hostConfig ClientHostConfig

	hostConfig.PolymorphicID = loginRes.PolymorphicID
	hostConfig.Current = true
	hostConfig.Host = host
	hostConfig.Name = loginRes.Name
	hostConfig.ProviderID = providerID
	hostConfig.AccessKey = loginRes.AccessKey
	hostConfig.SkipTLSVerify = skipTLSVerify

	var found bool

	for i, c := range config.Hosts {
		if c.Host == host {
			config.Hosts[i] = hostConfig
			found = true

			continue
		}

		config.Hosts[i].Current = false
	}

	if !found {
		config.Hosts = append(config.Hosts, hostConfig)
	}

	err = writeConfig(config)
	if err != nil {
		return err
	}

	client, err := apiClient(host, loginRes.AccessKey, skipTLSVerify)
	if err != nil {
		return err
	}

	if err := updateKubeconfig(client, loginRes.PolymorphicID); err != nil {
		return err
	}

	return nil
}

func isNonInteractiveMode() bool {
	return rootOptions.NonInteractive || os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))
}

func oidcflow(host string, clientId string) (string, error) {
	state, err := generate.CryptoRandom(12)
	if err != nil {
		return "", err
	}

	authorizeURL := fmt.Sprintf("https://%s/oauth2/v1/authorize?redirect_uri=http://localhost:8301&client_id=%s&response_type=code&scope=openid+email+groups+offline_access&state=%s", host, clientId, state)

	ls, err := newLocalServer()
	if err != nil {
		return "", err
	}

	err = browser.OpenURL(authorizeURL)
	if err != nil {
		return "", err
	}

	code, recvstate, err := ls.wait(time.Minute * 5)
	if err != nil {
		return "", err
	}

	if state != recvstate {
		//lint:ignore ST1005, user facing error
		return "", fmt.Errorf("Login aborted, provider state did not match the expected state")
	}

	return code, nil
}

// Prompt user to change their preset password when loggin in for the first time
func updateUserPassword(client *api.Client, pid uid.PolymorphicID, oldPassword string) error {
	// Todo otp: update term to temporary password (https://github.com/infrahq/infra/issues/1441)
	fmt.Println("\n  One time password was used.")

	newPassword, err := promptUpdatePassword(oldPassword)
	if err != nil {
		return err
	}

	userID, err := pid.ID()
	if err != nil {
		return fmt.Errorf("update user id login: %w", err)
	}

	if _, err := client.UpdateIdentity(&api.UpdateIdentityRequest{ID: userID, Password: newPassword}); err != nil {
		return fmt.Errorf("update user login: %w", err)
	}

	fmt.Println("  Password updated.")

	return nil
}

// Given the provider name, directs user to its OIDC login page, then saves the auth code (to later login to infra)
func loginToProviderN(client *api.Client, providerName string) (*api.LoginRequestOIDC, error) {
	provider, err := GetProviderByName(client, providerName)
	if err != nil {
		return nil, err
	}
	return loginToProvider(provider)
}

// Given the provider, directs user to its OIDC login page, then saves the auth code (to later login to infra)
func loginToProvider(provider *api.Provider) (*api.LoginRequestOIDC, error) {
	fmt.Fprintf(os.Stderr, "  Logging in with %s...\n", termenv.String(provider.Name).Bold().String())

	code, err := oidcflow(provider.URL, provider.ClientID)
	if err != nil {
		return nil, err
	}

	return &api.LoginRequestOIDC{
		ProviderID:  provider.ID,
		RedirectURL: cliLoginRedirectURL,
		Code:        code,
	}, nil
}

func runSetupForLogin(client *api.Client) (string, error) {
	setupRes, err := client.Setup()
	if err != nil {
		return "", err
	}

	fmt.Println()
	fmt.Printf("  Congratulations, Infra has been successfully installed.\n")
	fmt.Printf("  Running setup for the first time...\n\n")
	fmt.Printf("  Access Key: %s\n", setupRes.AccessKey)
	fmt.Printf(fmt.Sprintf("  %s", termenv.String("IMPORTANT: Store in a safe place. You will not see it again.\n\n").Bold().String()))

	return setupRes.AccessKey, nil
}

// Only used when logging in or switching to a new session, since user has no credentials. Otherwise, use defaultAPIClient().
func newAPIClient(server string, skipTLSVerify bool) (*api.Client, error) {
	if !skipTLSVerify {
		// Prompt user only if server fails the TLS verification
		if err := verifyTLS(server); err != nil {
			if !errors.Is(err, ErrTLSNotVerified) {
				return nil, err
			}

			if err = promptSkipTLSVerify(); err != nil {
				return nil, err
			}
			skipTLSVerify = true
		}
	}

	client, err := apiClient(server, "", skipTLSVerify)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func verifyTLS(host string) error {
	url, err := urlx.Parse(host)
	if err != nil {
		logging.S.Debug("Cannot parse host", host, err)
		logging.S.Error("Could not login. Please check the server hostname")
		return err
	}
	url.Scheme = "https"
	urlString := url.String()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, urlString, nil)
	if err != nil {
		logging.S.Debugf("Cannot create request: %v", err)
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if !errors.As(err, &x509.UnknownAuthorityError{}) && !errors.As(err, &x509.HostnameError{}) && !strings.Contains(err.Error(), "certificate is not trusted") {
			logging.S.Debugf("Cannot validate TLS due to an unexpected error: %v", err)
			return err
		}

		logging.S.Debug(err)

		return ErrTLSNotVerified
	}

	defer res.Body.Close()
	return nil
}

func promptLocalLogin() (*api.LoginRequestPasswordCredentials, error) {
	var credentials struct {
		Email    string
		Password string
	}

	questionPrompt := []*survey.Question{
		{
			Name:     "Email",
			Prompt:   &survey.Input{Message: "   Email:"},
			Validate: survey.Required,
		},
		{
			Name:     "Password",
			Prompt:   &survey.Password{Message: "Password:"},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(questionPrompt, &credentials, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return &api.LoginRequestPasswordCredentials{}, err
	}

	return &api.LoginRequestPasswordCredentials{
		Email:    credentials.Email,
		Password: credentials.Password,
	}, nil
}

func promptAccessKeyLogin() (string, error) {
	var accessKey string
	if err := survey.AskOne(&survey.Password{Message: "Access Key:"}, &accessKey, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}
	return accessKey, nil
}

func listProviders(client *api.Client) ([]api.Provider, error) {
	providers, err := client.ListProviders("")
	if err != nil {
		return nil, err
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Name < providers[j].Name
	})

	return providers, nil
}

func promptLoginOptions(client *api.Client) (loginMethod loginMethod, provider *api.Provider, err error) {
	if isNonInteractiveMode() {
		return 0, nil, fmt.Errorf("Non-interactive login requires key, instead run: 'infra login SERVER --non-interactive --key KEY")
	}

	providers, err := listProviders(client)
	if err != nil {
		return 0, nil, err
	}

	var options []string
	for _, p := range providers {
		options = append(options, fmt.Sprintf("%s (%s)", p.Name, p.URL))
	}

	options = append(options, "Login as a local user")
	options = append(options, "Login with an access key")

	var i int
	selectPrompt := &survey.Select{
		Message: "Select a login method:",
		Options: options,
	}
	err = survey.AskOne(selectPrompt, &i, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if errors.Is(err, terminal.InterruptErr) {
		return 0, nil, err
	}

	switch i {
	case len(options) - 1: // last option: accessKeyLogin
		return accessKeyLogin, nil, nil
	case len(options) - 2: // second last option: localLogin
		return localLogin, nil, nil
	default:
		return oidcLogin, &providers[i], nil
	}
}

// Error out if it fails TLS verification and user does not want to connect.
func promptSkipTLSVerify() error {
	if isNonInteractiveMode() {
		fmt.Fprintf(os.Stderr, "%s\n", ErrTLSNotVerified.Error())
		return fmt.Errorf("Non-interactive login does not allow insecure connection by default,\n       unless overridden with  '--skip-tls-verify'.")
	}

	// Although the same error, format is a little different for interactive/non-interactive.
	fmt.Fprintf(os.Stderr, "  %s\n", ErrTLSNotVerified.Error())
	confirmPrompt := &survey.Confirm{
		Message: "Are you sure you want to continue?",
	}
	proceed := false
	if err := survey.AskOne(confirmPrompt, &proceed, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return err
	}
	if !proceed {
		return terminal.InterruptErr
	}
	return nil
}

// Returns the host address of the Infra server that user would like to log into
func promptServer() (string, error) {
	if isNonInteractiveMode() {
		return "", fmt.Errorf("Non-interactive login requires the [SERVER] argument")
	}

	config, err := readOrCreateClientConfig()
	if err != nil {
		return "", err
	}

	hosts := config.HostNames()

	if len(hosts) == 0 {
		return promptNewHost()
	}

	return promptExistingHosts(hosts)
}

func promptNewHost() (string, error) {
	var host string
	err := survey.AskOne(&survey.Input{Message: "Host:"}, &host, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithValidator(survey.Required))
	if err != nil {
		return "", err
	}

	return host, nil
}

func promptExistingHosts(hosts []string) (string, error) {
	const defaultOption string = "Connect to a different host"
	hosts = append(hosts, defaultOption)

	prompt := &survey.Select{
		Message: "Select a server:",
		Options: hosts,
	}

	filter := func(filterValue string, optValue string, optIndex int) bool {
		return strings.Contains(optValue, filterValue) || strings.EqualFold(optValue, defaultOption)
	}

	var i int
	if err := survey.AskOne(prompt, &i, survey.WithFilter(filter), survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return "", err
	}

	if hosts[i] == defaultOption {
		return promptNewHost()
	}

	return hosts[i], nil
}
