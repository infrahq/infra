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

type loginOptions struct {
	Server        string `mapstructure:"server"`
	AccessKey     string `mapstructure:"key"`
	Provider      string `mapstructure:"provider"`
	SkipTLSVerify bool   `mapstructure:"skipTLSVerify"`
}

const cliLoginRedirectURL = "http://localhost:8301"

func login(options loginOptions) error {
	if options.Server == "" {
		if isNonInteractiveMode() {
			return errors.New(`Non-interactive login requires the [SERVER] argument`)
		}

		var err error
		if options.Server, err = promptHost(); err != nil {
			return err
		}
	}

	client, err := getAPIClient(options.Server, &options.SkipTLSVerify)
	if err != nil {
		return err
	}

	// Check if setup is required. If so, user will automatically be logged in with the setup accessKey.
	setupRequired, err := client.SetupRequired()
	if err != nil {
		return err
	}
	if setupRequired.Required {
		if options.AccessKey != "" {
			return errors.New(`Infra has not been setup. To setup, run the following without any additional args: 'infra login [SERVER]'`)
		}
		if options.AccessKey, err = runSetupForLogin(client); err != nil {
			return err
		}

	}

	loginReq := &api.LoginRequest{}
	var provider api.Provider

	switch {
	case options.AccessKey != "":
		loginReq.AccessKey = options.AccessKey
	case options.Provider != "":
		provider, err := GetProviderByName(client, options.Provider)
		if err != nil {
			return err
		}
		if err = loginToProvider(loginReq, *provider); err != nil {
			return err
		}
	default:
		if isNonInteractiveMode() {
			return errors.New(`Non-interactive login requires key, instead run: 'infra login SERVER --key KEY'`)
		}

		providers, err := client.ListProviders("")
		if err != nil {
			return err
		}

		provider, err = promptLoginOptions(providers, loginReq)
		if err != nil {
			return err
		}
	}

	loginRes, err := client.Login(loginReq)
	if err != nil {
		return err
	}

	return finishLogin(options.Server, options.SkipTLSVerify, provider.ID, loginRes)
}

func relogin() error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if isNonInteractiveMode() {
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
		return errors.New("Cannot renew login without provider")
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
		return "", errors.New("Login aborted, provider state did not match the expected state")
	}

	return code, nil
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

// Directs user to OIDC login page, then saves the auth code (to later login to infra)
func loginToProvider(loginReq *api.LoginRequest, provider api.Provider) error {
	fmt.Fprintf(os.Stderr, "  Logging in with %s...\n", termenv.String(provider.Name).Bold().String())

	code, err := oidcflow(provider.URL, provider.ClientID)
	if err != nil {
		return err
	}

	loginReq.OIDC = &api.LoginRequestOIDC{
		ProviderID:  provider.ID,
		RedirectURL: cliLoginRedirectURL,
		Code:        code,
	}
	return nil
}

func runSetupForLogin(client *api.Client) (string, error) {
	setupRes, err := client.Setup()
	if err != nil {
		return "", err
	}

	fmt.Println()
	fmt.Printf("  Congratulations, Infra has been successfully installed.\n\n")
	fmt.Printf("  Access Key: %s\n", setupRes.AccessKey)
	fmt.Printf(fmt.Sprintf("  %s", termenv.String("IMPORTANT: Store in a safe place. You will not see it again.\n\n").Bold().String()))

	return setupRes.AccessKey, nil
}

func getAPIClient(host string, skipTLSVerify *bool) (*api.Client, error) {
	if !*skipTLSVerify {
		if err := verifyTLS(host); err != nil {
			if !errors.Is(err, ErrTLS) {
				return nil, err
			}

			if isNonInteractiveMode() {
				return nil, errors.New(ErrTLS.Error() + "\nTo continue with the insecure connection, run with '--skip-tls-verify'")
			}

			if err = promptSkipTLSVerify(); err != nil {
				return nil, err
			}
			*skipTLSVerify = true
		}
	}
	return apiClient(host, "", *skipTLSVerify)
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
		logging.S.Debug("Cannot create request", err)
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if !errors.As(err, &x509.UnknownAuthorityError{}) && !errors.As(err, &x509.HostnameError{}) && !strings.Contains(err.Error(), "certificate is not trusted") {
			logging.S.Debug("Cannot validate TLS due to an unexpected error", err)
			return err
		}

		logging.S.Debug(err)

		return ErrTLS
	}

	defer res.Body.Close()
	return nil
}

func promptLocalLogin(loginReq *api.LoginRequest) error {
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

func promptAccessKeyLogin(loginReq *api.LoginRequest) error {
	if err := survey.AskOne(&survey.Password{Message: "Access Key:"}, &loginReq.AccessKey, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		return err
	}
	return nil
}

func promptLoginOptions(providers []api.Provider, loginReq *api.LoginRequest) (api.Provider, error) {
	var options []string
	var provider api.Provider

	for _, p := range providers {
		if p.Name == models.InternalInfraProviderName {
			options = append(options, "Email and Password")
		} else {
			options = append(options, fmt.Sprintf("%s (%s)", p.Name, p.URL))
		}
	}

	options = append(options, "Login with Access Key")

	var option int
	prompt := &survey.Select{
		Message: "Select a login method:",
		Options: options,
	}

	err := survey.AskOne(prompt, &option, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if errors.Is(err, terminal.InterruptErr) {
		return provider, err
	}

	switch {
	case option == len(providers):
		if err = promptAccessKeyLogin(loginReq); err != nil {
			return provider, err
		}
	case providers[option].Name == models.InternalInfraProviderName:
		if err = promptLocalLogin(loginReq); err != nil {
			return provider, err
		}
	default:
		if err = loginToProvider(loginReq, providers[option]); err != nil {
			return provider, err
		}
		provider = providers[option]
	}

	return provider, nil
}

func promptSkipTLSVerify() error {
	prompt := &survey.Confirm{
		Message: ErrTLS.Error() + "\n  Are you sure you want to continue?",
	}
	proceed := false
	if err := survey.AskOne(prompt, &proceed, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)); err != nil {
		logging.S.Debug(err)
		return err
	}
	if !proceed {
		return ErrTLS
	}
	return nil
}

// Returns the host address of the Infra server that user would like to log into
func promptHost() (string, error) {
	config, err := readOrCreateConfig()
	if err != nil {
		return "", err
	}

	hosts := config.getHostsStr()

	if len(hosts) == 0 {
		return promptNewHost()
	}

	return promptExistingHosts(hosts)
}

func promptNewHost() (string, error) {
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
