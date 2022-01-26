package cmd

import (
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
	"github.com/muesli/termenv"
	"golang.org/x/term"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
)

type LoginOptions struct {
	Current          bool          `mapstructure:"current"`
	Timeout          time.Duration `mapstructure:"timeout"`
	internal.Options `mapstructure:",squash"`
}

func login(options *LoginOptions) error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return ErrConfigNotFound
	}

	loadedCfg, err := readConfig()
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return err
	}

	if loadedCfg == nil {
		loadedCfg = NewClientConfig()
	}

	var selectedHost *ClientHostConfig

HOST:
	switch {
	case options.Host == "":
		if options.Current {
			for i := range loadedCfg.Hosts {
				if loadedCfg.Hosts[i].Current {
					selectedHost = &loadedCfg.Hosts[i]
					break HOST
				}
			}
		}

		if len(loadedCfg.Hosts) > 0 {
			selectedHost, err = promptSelectHost(loadedCfg.Hosts)
			if err != nil {
				return err
			}

			if selectedHost != nil {
				break HOST
			}
		}

		err := survey.AskOne(&survey.Input{Message: "Host"}, &options.Host, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
		if err != nil {
			if errors.Is(err, terminal.InterruptErr) {
				return nil
			}

			return err
		}

		fallthrough
	default:
		for i := range loadedCfg.Hosts {
			if loadedCfg.Hosts[i].Host == options.Host {
				selectedHost = &loadedCfg.Hosts[i]
				break HOST
			}
		}

		loadedCfg.Hosts = append(loadedCfg.Hosts, ClientHostConfig{
			Host:    options.Host,
			Current: true,
		})
		selectedHost = &loadedCfg.Hosts[len(loadedCfg.Hosts)-1]
	}

	if selectedHost == nil {
		//lint:ignore ST1005, user facing error
		return errors.New("Host endpoint is required, ask your administrator for the endpoint you should use to login")
	}

	fmt.Fprintf(os.Stderr, "  Logging in to %s\n", termenv.String(selectedHost.Host).Bold().String())

	skipTLSVerify := selectedHost.SkipTLSVerify
	if !skipTLSVerify {
		var proceed bool

		skipTLSVerify, proceed, err = promptShouldSkipTLSVerify(selectedHost.Host)
		if err != nil {
			return err
		}

		if !proceed {
			//lint:ignore ST1005, user facing error
			return fmt.Errorf("Login cancelled, not proceeding with TLS connection that could not be verified")
		}
	}

	client, err := apiClient(selectedHost.Host, "", skipTLSVerify)
	if err != nil {
		return err
	}

	providers, err := client.ListProviders()
	if err != nil {
		return err
	}

	var selectedProvider *api.Provider

provider:
	switch {
	case len(providers) == 0:
		//lint:ignore ST1005, user facing error
		return errors.New("No identity providers have been configured for logging in with this Infra host")
	case len(providers) == 1:
		selectedProvider = &providers[0]
	default:
		// Use the current provider ID if it's valid to avoid prompting the user
		if selectedHost.ProviderID != "" && options.Current {
			for i, provider := range providers {
				if provider.ID == selectedHost.ProviderID {
					selectedProvider = &providers[i]
					break provider
				}
			}
		}

		selectedProvider, err = promptSelectProvider(providers)
		if errors.Is(err, terminal.InterruptErr) {
			return nil
		}

		if err != nil {
			return err
		}
	}

	var loginReq api.LoginRequest

	switch {
	case selectedProvider.Kind == "okta":
		// Start OIDC flow
		// Get auth code from Okta
		// Send auth code to Infra to login as a user
		state, err := generate.CryptoRandom(12)
		if err != nil {
			return err
		}

		authorizeURL := fmt.Sprintf("https://%s/oauth2/v1/authorize?redirect_uri=http://localhost:8301&client_id=%s&response_type=code&scope=openid+email+groups+offline_access&state=%s", selectedProvider.Domain, selectedProvider.ClientID, state)

		fmt.Fprintf(os.Stderr, "  Logging in with %s...\n", termenv.String("Okta").Bold().String())

		ls, err := newLocalServer()
		if err != nil {
			return err
		}

		err = browser.OpenURL(authorizeURL)
		if err != nil {
			return err
		}

		code, recvstate, err := ls.wait(options.Timeout)
		if err != nil {
			return err
		}

		if state != recvstate {
			//lint:ignore ST1005, user facing error
			return errors.New("Login aborted, Okta state did not match the expected state")
		}

		loginReq.Okta = &api.LoginRequestOkta{
			Domain: selectedProvider.Domain,
			Code:   code,
		}
	default:
		//lint:ignore ST1005, user facing error, should not happen
		return fmt.Errorf("Invalid provider selected %q", selectedProvider.Kind)
	}

	loginRes, err := client.Login(&loginReq)
	if err != nil {
		return err
	}

	for i := range loadedCfg.Hosts {
		loadedCfg.Hosts[i].Current = false
	}

	selectedHost.Name = loginRes.Name
	selectedHost.Token = loginRes.Token
	selectedHost.SkipTLSVerify = skipTLSVerify
	selectedHost.ProviderID = selectedProvider.ID
	selectedHost.Current = true

	err = writeConfig(loadedCfg)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  Logged in as %s\n", termenv.String(loginRes.Name).Bold().String())

	client, err = apiClient(selectedHost.Host, selectedHost.Token, selectedHost.SkipTLSVerify)
	if err != nil {
		return err
	}

	users, err := client.ListUsers(loginRes.Name)
	if err != nil {
		return err
	}

	if len(users) < 1 {
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("User %q not found at Infra host, is this account still valid?", loginRes.Name)
	}

	if len(users) > 1 {
		//lint:ignore ST1005, user facing error
		return fmt.Errorf("Found multiple users found for %q, please contact your administrator", loginRes.Name)
	}

	err = updateKubeconfig(users[0])
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if len(users[0].Grants) > 0 {
		kubeConfigFilename := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ConfigAccess().GetDefaultFilename()
		fmt.Fprintf(os.Stderr, "  Updated %s\n", termenv.String(strings.ReplaceAll(kubeConfigFilename, homeDir, "~")).Bold().String())
	}

	context, err := switchToFirstInfraContext()
	if err != nil {
		return err
	}

	if context != "" {
		fmt.Fprintf(os.Stderr, "  Current Kubernetes context is now %s\n", termenv.String(context).Bold().String())
	}

	return nil
}

func promptSelectHost(hosts []ClientHostConfig) (*ClientHostConfig, error) {
	options := []string{}
	for _, reg := range hosts {
		options = append(options, reg.Host)
	}

	options = append(options, "Connect to a different host")

	option := 0
	prompt := &survey.Select{
		Message: "Select an Infra host:",
		Options: options,
	}

	err := survey.AskOne(prompt, &option, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if err != nil {
		return nil, err
	}

	if option == len(options)-1 {
		return nil, nil
	}

	return &hosts[option], nil
}

func promptShouldSkipTLSVerify(host string) (shouldSkipTLSVerify bool, proceed bool, err error) {
	url, err := urlx.Parse(host)
	if err != nil {
		return false, false, fmt.Errorf("parsing host: %w", err)
	}

	url.Scheme = "https"
	urlString := url.String()

	req, err := http.NewRequest(http.MethodGet, urlString, nil)
	if err != nil {
		return false, false, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if !errors.Is(err, x509.CertificateInvalidError{}) && !errors.Is(err, x509.SystemRootsError{}) && !strings.Contains(err.Error(), "certificate is not trusted") {
			return false, false, err
		}

		proceed := false

		fmt.Fprintf(os.Stderr, "Could not verify certificate for host %q: %s\n", host, err)

		prompt := &survey.Confirm{
			Message: "Are you sure you want to continue?",
		}

		err := survey.AskOne(prompt, &proceed, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
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

func promptSelectProvider(providers []api.Provider) (*api.Provider, error) {
	if providers == nil {
		return nil, errors.New("providers cannot be nil")
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Created > providers[j].Created
	})

	options := []string{}

	for _, p := range providers {
		if p.Kind == "okta" {
			options = append(options, fmt.Sprintf("Okta [%s]", p.Domain))
		}
	}

	var option int

	prompt := &survey.Select{
		Message: "Select a login method:",
		Options: options,
	}

	err := survey.AskOne(prompt, &option, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if errors.Is(err, terminal.InterruptErr) {
		return nil, err
	}

	return &providers[option], nil
}
