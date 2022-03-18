package cmd

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/goware/urlx"
	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

const cliLoginRedirectURL = "http://localhost:8301"

func relogin() error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if !isInteractiveMode() {
		return errors.New("Non-interactive login is not supported")
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
		return errors.New("can not renew login without provider")
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

func isInteractiveMode() bool {
	if rootOptions.NonInteractive {
		// user explicitly asked for a non-interactive terminal
		return false
	}

	if os.Stdin == nil {
		return false
	}

	return term.IsTerminal(int(os.Stdin.Fd()))
}

func login(host string) error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if !isInteractiveMode() {
		return errors.New("Non-interactive login is not supported")
	}

	config, err := readConfig()
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return err
	}

	if config == nil {
		config = NewClientConfig()
	}

	if host == "" {
		var hosts []string
		for _, h := range config.Hosts {
			hosts = append(hosts, h.Host)
		}

		host, err = promptHost(hosts)
		if err != nil {
			return err
		}
	}

	u, err := urlx.Parse(host)
	if err != nil {
		return err
	}

	host = u.Host

	fmt.Fprintf(os.Stderr, "  Logging in to %s\n", termenv.String(host).Bold().String())

	skipTLSVerify, proceed, err := promptShouldSkipTLSVerify(host)
	if err != nil {
		return err
	}

	if !proceed {
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("Could not verify TLS connection")
	}

	client, err := apiClient(host, "", skipTLSVerify)
	if err != nil {
		return err
	}

	// get the initial access key if this is a new Infra deploy
	accessKey, err := checkDoSetup(client)
	if err != nil {
		return err
	}

	// setup the authentication options Infra supports (OIDC, local, access key exchange, etc.)
	options, oidcProviders, providerID, err := authenticationOptions(client)
	if err != nil {
		return err
	}

	var option int

	if len(options) > 1 {
		option, err = promptProvider(options)
		if err != nil {
			return err
		}
	}

	loginReq := &api.LoginRequest{}

	switch option {
	case len(options) - 1:
		if err := setupAccessKeyExchangeLogin(loginReq, accessKey); err != nil {
			return err
		}
	case len(options) - 2:
		if err := setupEmailAndPasswordLogin(loginReq); err != nil {
			return err
		}
	default:
		if providerID, err = getOIDCAuthCode(loginReq, oidcProviders, option); err != nil {
			return err
		}
	}

	loginRes, err := client.Login(loginReq)
	if err != nil {
		return err
	}

	return finishLogin(host, skipTLSVerify, providerID, loginRes)
}

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

	if loginRes.PasswordUpdateRequired {
		if err := updateUserPassword(client, loginRes.PolymorphicID); err != nil {
			return err
		}
	}

	return nil
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
		return "", errors.New("Login aborted, provider state did not match the expected state")
	}

	return code, nil
}

func promptProvider(providers []string) (int, error) {
	var option int

	prompt := &survey.Select{
		Message: "Select a login method:",
		Options: providers,
	}

	err := survey.AskOne(prompt, &option, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if errors.Is(err, terminal.InterruptErr) {
		return 0, err
	}

	return option, nil
}

func promptHost(hosts []string) (string, error) {
	var option int

	const defaultOpt string = "Connect to a different host"

	hosts = append(hosts, defaultOpt)

	if len(hosts) > 0 {
		prompt := &survey.Select{
			Message: "Select an Infra host:",
			Options: hosts,
		}

		filter := func(filterValue string, optValue string, optIndex int) bool {
			return strings.Contains(optValue, filterValue) || strings.EqualFold(optValue, defaultOpt)
		}

		err := survey.AskOne(prompt, &option, survey.WithFilter(filter), survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
		if err != nil {
			return "", err
		}
	}

	if option != len(hosts)-1 {
		return hosts[option], nil
	}

	var host string

	err := survey.AskOne(&survey.Input{Message: "Host:"}, &host, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if err != nil {
		return "", err
	}

	if host == "" {
		return "", errors.New("Host is required")
	}

	return host, nil
}

func promptShouldSkipTLSVerify(host string) (shouldSkipTLSVerify bool, proceed bool, err error) {
	url, err := urlx.Parse(host)
	if err != nil {
		return false, false, fmt.Errorf("parsing host: %w", err)
	}

	url.Scheme = "https"
	urlString := url.String()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, urlString, nil)
	if err != nil {
		return false, false, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if !errors.As(err, &x509.UnknownAuthorityError{}) && !errors.As(err, &x509.HostnameError{}) && !strings.Contains(err.Error(), "certificate is not trusted") {
			return false, false, err
		}

		logging.S.Debug(err)

		fmt.Printf("  The authenticity of host '%s' can't be established.\n", host)

		prompt := &survey.Confirm{
			Message: "Are you sure you want to continue?",
		}

		proceed := false

		err := survey.AskOne(prompt, &proceed, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
		if err != nil {
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

// authenticationOptions returns the ways to login based on the configuration of Infra
func authenticationOptions(client *api.Client) ([]string, []api.Provider, uid.ID, error) {
	providers, err := client.ListProviders("")
	if err != nil {
		return nil, nil, 0, err
	}

	localUsersEnabled := false

	options := []string{}
	oidcProviders := []api.Provider{}
	defaultProvider := uid.ID(0)

	for _, p := range providers {
		if p.Name == models.InternalInfraProviderName {
			localUsersEnabled = true
			defaultProvider = p.ID // default to the local infra provider, if something else is selected it will update
		} else {
			options = append(options, fmt.Sprintf("%s (%s)", p.Name, p.URL))
			oidcProviders = append(oidcProviders, p)
		}
	}

	if localUsersEnabled {
		// this is separate so that it appears as the bottom of the list
		options = append(options, "Email and Password")
	}

	options = append(options, "Login with Access Key")

	return options, oidcProviders, defaultProvider, nil
}

// checkDoSetup gets the initial admin access key if Infra setup is required
func checkDoSetup(client *api.Client) (string, error) {
	setupRequired, err := client.SetupRequired()
	if err != nil {
		return "", err
	}

	if setupRequired.Required {
		setup, err := client.Setup()
		if err != nil {
			return "", err
		}

		fmt.Println()
		fmt.Printf("  Congratulations, Infra has been successfully installed.\n\n")
		fmt.Printf("  Access Key: %s\n", setup.AccessKey)
		fmt.Printf(fmt.Sprintf("  %s", termenv.String("IMPORTANT: Store in a safe place. You will not see it again.\n\n").Bold().String()))

		return setup.AccessKey, nil
	}

	return "", nil
}

// setupAccessKeyExchangeLogin prompts for the access key to login to Infra
func setupAccessKeyExchangeLogin(loginReq *api.LoginRequest, accessKey string) error {
	if accessKey == "" {
		if err := survey.AskOne(&survey.Password{Message: "Access Key:"}, &accessKey, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
			return err
		}
	}

	loginReq.AccessKey = accessKey

	return nil
}

// setupEmailAndPasswordLogin prompts for the username and password to login to Infra
func setupEmailAndPasswordLogin(loginReq *api.LoginRequest) error {
	fmt.Println("  Logging in with email and password...")

	var credentials struct {
		Email    string
		Password string
	}

	emailPassPrompt := []*survey.Question{
		{
			Name:     "Email",
			Prompt:   &survey.Input{Message: "Email: "},
			Validate: survey.Required,
		},
		{
			Name:     "Password",
			Prompt:   &survey.Password{Message: "Password: "},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(emailPassPrompt, &credentials, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return err
	}

	loginReq.PasswordCredentials = &api.LoginRequestPasswordCredentials{
		Email:    credentials.Email,
		Password: credentials.Password,
	}

	return nil
}

func getOIDCAuthCode(loginReq *api.LoginRequest, oidcProviders []api.Provider, option int) (uid.ID, error) {
	provider := oidcProviders[option]

	fmt.Fprintf(os.Stderr, "  Logging in with %s...\n", termenv.String(provider.Name).Bold().String())

	code, err := oidcflow(provider.URL, provider.ClientID)
	if err != nil {
		return uid.ID(0), err
	}

	loginReq.OIDC = &api.LoginRequestOIDC{
		ProviderID:  provider.ID,
		RedirectURL: cliLoginRedirectURL,
		Code:        code,
	}

	return provider.ID, nil
}

// updateUserPassword sets the user password after a one time password is used to login
func updateUserPassword(client *api.Client, userPID uid.PolymorphicID) error {
	newPassword := ""
	if err := survey.AskOne(&survey.Password{Message: "One time password used, please set a new password:"}, &newPassword, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return err
	}

	userID, err := userPID.ID()
	if err != nil {
		return fmt.Errorf("update user id login: %w", err)
	}

	if _, err := client.UpdateUser(&api.UpdateUserRequest{ID: userID, Password: newPassword}); err != nil {
		return fmt.Errorf("update user login: %w", err)
	}

	fmt.Println("  Password updated, you're all set")

	return nil
}
