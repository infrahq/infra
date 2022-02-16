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

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/uid"
)

func relogin() error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if !term.IsTerminal(int(os.Stdin.Fd())) {
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
		ProviderID: provider.ID,
		Code:       code,
	}

	loginRes, err := client.Login(loginReq)
	if err != nil {
		return err
	}

	return finishLogin(currentConfig.Host, loginRes.ID, loginRes.Name, loginRes.Token, currentConfig.SkipTLSVerify, 0)
}

func login(host string) error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if !term.IsTerminal(int(os.Stdin.Fd())) {
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

	providers, err := client.ListProviders("")
	if err != nil {
		return err
	}

	var options []string
	for _, p := range providers {
		options = append(options, fmt.Sprintf("%s (%s)", p.Name, p.URL))
	}

	options = append(options, "Login with Access Key")

	option, err := promptProvider(options)
	if err != nil {
		return err
	}

	// access key
	if option == len(options)-1 {
		var token string

		err = survey.AskOne(&survey.Password{Message: "Your Access Key:"}, &token, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
		if err != nil {
			return err
		}

		return finishLogin(host, 0, "system", token, skipTLSVerify, 0)
	}

	provider := providers[option]

	fmt.Fprintf(os.Stderr, "  Logging in with %s...\n", termenv.String(provider.Name).Bold().String())

	code, err := oidcflow(provider.URL, provider.ClientID)
	if err != nil {
		return err
	}

	loginReq := &api.LoginRequest{
		ProviderID: provider.ID,
		Code:       code,
	}

	loginRes, err := client.Login(loginReq)
	if err != nil {
		return err
	}

	return finishLogin(host, loginRes.ID, loginRes.Name, loginRes.Token, skipTLSVerify, provider.ID)
}

func finishLogin(host string, id uid.ID, name string, token string, skipTLSVerify bool, providerID uid.ID) error {
	client, err := apiClient(host, token, skipTLSVerify)
	if err != nil {
		return err
	}

	_, err = client.ListUsers(api.ListUsersRequest{Email: name})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  Logged in as %s\n", termenv.String(name).Bold().String())

	config, err := readConfig()
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return err
	}

	if config == nil {
		config = NewClientConfig()
	}

	var hostConfig ClientHostConfig

	hostConfig.ID = id
	hostConfig.Current = true
	hostConfig.Host = host
	hostConfig.Name = name
	hostConfig.ProviderID = providerID
	hostConfig.Token = token
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

	if id != 0 {
		return updateKubeconfig(client, id)
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

type hostchoice struct {
	Host string
	Auto bool
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

		proceed := false

		fmt.Fprintf(os.Stderr, "Could not verify certificate for host %q: %s\n", host, err)

		prompt := &survey.Confirm{
			Message: "Are you sure you want to continue?",
		}

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
