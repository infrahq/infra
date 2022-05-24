package cmd

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/goware/urlx"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
)

type loginCmdOptions struct {
	Server         string
	AccessKey      string
	Provider       string
	SkipTLSVerify  bool
	NonInteractive bool
}

type loginMethod int8

const (
	localLogin loginMethod = iota
	accessKeyLogin
	oidcLogin
)

const cliLoginRedirectURL = "http://localhost:8301"

func newLoginCmd(_ *CLI) *cobra.Command {
	var options loginCmdOptions

	cmd := &cobra.Command{
		Use:   "login [SERVER]",
		Short: "Login to Infra",
		Example: `# By default, login will prompt for all required information.
$ infra login

# Login to a specific server
$ infra login infraexampleserver.com

# Login with a specific identity provider
$ infra login --provider okta

# Login with an access key
$ infra login --key 1M4CWy9wF5.fAKeKEy5sMLH9ZZzAur0ZIjy`,
		Args:  MaxArgs(1),
		Group: "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				options.Server = args[0]
			}

			return login(options)
		},
	}

	cmd.Flags().StringVar(&options.AccessKey, "key", "", "Login with an access key")
	cmd.Flags().StringVar(&options.Provider, "provider", "", "Login with an identity provider")
	cmd.Flags().BoolVar(&options.SkipTLSVerify, "skip-tls-verify", false, "Skip verifying server TLS certificates")
	addNonInteractiveFlag(cmd.Flags(), &options.NonInteractive)
	return cmd
}

func login(options loginCmdOptions) error {
	var err error

	if options.Server == "" {
		options.Server, err = promptServer(options)
		if err != nil {
			return err
		}
	}

	client, err := newAPIClient(options)
	if err != nil {
		return err
	}

	loginReq := &api.LoginRequest{}

	// if signup is required, use it to create an admin account
	// and use those credentials for subsequent requests
	signupEnabled, err := client.SignupEnabled()
	if err != nil {
		return err
	}

	if signupEnabled.Enabled {
		loginReq.PasswordCredentials, err = runSignupForLogin(client)
		if err != nil {
			return err
		}

		return loginToInfra(client, loginReq)
	}

	switch {
	case options.AccessKey != "":
		loginReq.AccessKey = options.AccessKey
	case options.Provider != "":
		loginReq.OIDC, err = loginToProviderN(client, options.Provider)
		if err != nil {
			return err
		}
	default:
		if options.NonInteractive {
			return fmt.Errorf("Non-interactive login requires key, instead run: 'infra login SERVER --non-interactive --key KEY")
		}
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

func loginToInfra(client *api.Client, loginReq *api.LoginRequest) error {
	loginRes, err := client.Login(loginReq)
	if err != nil {
		if api.ErrorStatusCode(err) == http.StatusUnauthorized || api.ErrorStatusCode(err) == http.StatusNotFound {
			switch {
			case loginReq.AccessKey != "":
				return &LoginError{Message: "your access key may be invalid"}
			case loginReq.PasswordCredentials != nil:
				return &LoginError{Message: "your username or password may be invalid"}
			case loginReq.OIDC != nil:
				return &LoginError{Message: "please contact an administrator and check identity provider configurations"}
			}
		}

		return err
	}

	if loginRes.PasswordUpdateRequired {
		fmt.Fprintf(os.Stderr, "  Your password has expired. Please update your password (min. length 8).\n")

		password, err := promptSetPassword(loginReq.PasswordCredentials.Password)
		if err != nil {
			return err
		}

		client.AccessKey = loginRes.AccessKey
		if _, err := client.UpdateUser(&api.UpdateUserRequest{ID: loginRes.UserID, Password: password}); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "  Updated password.\n")
	}

	if err := updateInfraConfig(client, loginReq, loginRes); err != nil {
		return err
	}

	// Client needs to be refreshed from here onwards, based on the newly saved infra configuration.
	client, err = defaultAPIClient()
	if err != nil {
		return err
	}

	if err := updateKubeconfig(client, loginRes.UserID); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  Logged in as %s\n", termenv.String(loginRes.Name).Bold().String())

	return nil
}

// Updates all configs with the current logged in session
func updateInfraConfig(client *api.Client, loginReq *api.LoginRequest, loginRes *api.LoginResponse) error {
	clientHostConfig := ClientHostConfig{
		Current:       true,
		PolymorphicID: uid.NewIdentityPolymorphicID(loginRes.UserID),
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

func oidcflow(host string, clientId string) (string, error) {
	// find out what the authorization endpoint is
	provider, err := oidc.NewProvider(context.Background(), fmt.Sprintf("https://%s", host))
	if err != nil {
		return "", fmt.Errorf("get provider oidc info: %w", err)
	}

	// claims are the attributes of the user we want to know from the identity provider
	var claims struct {
		ScopesSupported []string `json:"scopes_supported"`
	}

	if err := provider.Claims(&claims); err != nil {
		return "", fmt.Errorf("parsing claims: %w", err)
	}

	scopes := []string{"openid", "email"} // openid and email are required scopes for login to work

	// we want to be able to use these scopes to access groups, but they are not needed
	wantScope := map[string]bool{
		"groups":         true,
		"offline_access": true,
	}

	for _, scope := range claims.ScopesSupported {
		if wantScope[scope] {
			scopes = append(scopes, scope)
		}
	}

	// the state makes sure we are getting the correct response for our request
	state, err := generate.CryptoRandom(12)
	if err != nil {
		return "", err
	}

	authorizeURL := fmt.Sprintf("%s?redirect_uri=http://localhost:8301&client_id=%s&response_type=code&scope=%s&state=%s", provider.Endpoint().AuthURL, clientId, strings.Join(scopes, "+"), state)

	// the local server receives the response from the identity provider and sends it along to the infra server
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

func runSignupForLogin(client *api.Client) (*api.LoginRequestPasswordCredentials, error) {
	fmt.Fprintln(os.Stderr, "  Welcome to Infra. Set up your admin user:")

	username := ""
	if err := survey.AskOne(&survey.Input{Message: "Username:"}, &username, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	password, err := promptSetPassword("")
	if err != nil {
		return nil, err
	}

	_, err = client.Signup(&api.SignupRequest{Name: username, Password: password})
	if err != nil {
		return nil, err
	}

	return &api.LoginRequestPasswordCredentials{
		Name:     username,
		Password: password,
	}, nil
}

// Only used when logging in or switching to a new session, since user has no credentials. Otherwise, use defaultAPIClient().
func newAPIClient(options loginCmdOptions) (*api.Client, error) {
	if !options.SkipTLSVerify {
		// Prompt user only if server fails the TLS verification
		if err := verifyTLS(options.Server); err != nil {
			urlErr := &url.Error{}
			if errors.As(err, &urlErr) {
				if urlErr.Timeout() {
					return nil, fmt.Errorf("%w: %s", api.ErrTimeout, err)
				}
			}

			if !errors.Is(err, ErrTLSNotVerified) {
				return nil, err
			}

			if options.NonInteractive {
				fmt.Fprintf(os.Stderr, "%s\n", ErrTLSNotVerified.Error())
				return nil, fmt.Errorf("Non-interactive login does not allow insecure connection by default,\n       unless overridden with  '--skip-tls-verify'.")
			}

			if err = promptSkipTLSVerify(); err != nil {
				return nil, err
			}
			options.SkipTLSVerify = true
		}
	}

	client, err := apiClient(options.Server, "", options.SkipTLSVerify)
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

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, urlString, nil)
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
		Username string
		Password string
	}

	questionPrompt := []*survey.Question{
		{
			Name:     "Username",
			Prompt:   &survey.Input{Message: "Username:"},
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
		Name:     credentials.Username,
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

	sort.Slice(providers.Items, func(i, j int) bool {
		return providers.Items[i].Name < providers.Items[j].Name
	})

	return providers.Items, nil
}

func promptLoginOptions(client *api.Client) (loginMethod loginMethod, provider *api.Provider, err error) {
	providers, err := listProviders(client)
	if err != nil {
		return 0, nil, err
	}

	var options []string
	for _, p := range providers {
		options = append(options, fmt.Sprintf("%s (%s)", p.Name, p.URL))
	}

	options = append(options, "Login with username and password")
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
func promptServer(options loginCmdOptions) (string, error) {
	if options.NonInteractive {
		return "", fmt.Errorf("Non-interactive login requires the [SERVER] argument")
	}

	config, err := readOrCreateClientConfig()
	if err != nil {
		return "", err
	}

	servers := config.Hosts

	if len(servers) == 0 {
		return promptNewServer()
	}

	return promptServerList(servers)
}

func promptNewServer() (string, error) {
	var server string
	err := survey.AskOne(&survey.Input{Message: "Server:"}, &server, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithValidator(survey.Required))
	if err != nil {
		return "", err
	}

	return server, nil
}

func promptServerList(servers []ClientHostConfig) (string, error) {
	var promptOptions []string
	for _, server := range servers {
		promptOptions = append(promptOptions, server.Host)
	}

	defaultOption := "Connect to a new server"
	promptOptions = append(promptOptions, defaultOption)

	prompt := &survey.Select{
		Message: "Select a server:",
		Options: promptOptions,
	}

	filter := func(filterValue string, optValue string, optIndex int) bool {
		return strings.Contains(optValue, filterValue) || strings.EqualFold(optValue, defaultOption)
	}

	var i int
	if err := survey.AskOne(prompt, &i, survey.WithFilter(filter), survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return "", err
	}

	if promptOptions[i] == defaultOption {
		return promptNewServer()
	}

	return servers[i].Host, nil
}
