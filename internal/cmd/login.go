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
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/muesli/termenv"
	"golang.org/x/term"
	"k8s.io/client-go/tools/clientcmd"
)

type LoginOptions struct {
	Host    string
	Current bool
	Timeout time.Duration
}

func login(options LoginOptions) error {
	// TODO (https://github.com/infrahq/infra/issues/488): support non-interactive login
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return &ErrConfigNotFound{}
	}

	loadedCfg, err := readConfig()
	if err != nil && !errors.Is(err, &ErrConfigNotFound{}) {
		return err
	}

	if loadedCfg == nil {
		loadedCfg = NewClientConfig()
	}

	var selectedRegistry *ClientRegistryConfig

registry:
	switch {
	case options.Host == "":
		if options.Current {
			for i := range loadedCfg.Registries {
				if loadedCfg.Registries[i].Current {
					selectedRegistry = &loadedCfg.Registries[i]
					break registry
				}
			}
		}

		// TODO (https://github.com/infrahq/infra/issues/496): prompt user instead of assuming the first registry
		// since they may not know where they are logging into
		if len(loadedCfg.Registries) == 1 {
			selectedRegistry = &loadedCfg.Registries[0]
			break
		}

		selectedRegistry = promptSelectRegistry(loadedCfg.Registries)
	default:
		for i := range loadedCfg.Registries {
			if loadedCfg.Registries[i].Host == options.Host {
				selectedRegistry = &loadedCfg.Registries[i]
				break registry
			}
		}

		loadedCfg.Registries = append(loadedCfg.Registries, ClientRegistryConfig{
			Host:    options.Host,
			Current: true,
		})
		selectedRegistry = &loadedCfg.Registries[len(loadedCfg.Registries)-1]
	}

	if selectedRegistry == nil {
		return errors.New("A registry endpoint is required to continue with login.")
	}

	fmt.Fprintf(os.Stderr, "%s Logging in to %s\n", blue("✓"), termenv.String(selectedRegistry.Host).Bold().String())

	skipTLSVerify := selectedRegistry.SkipTLSVerify
	if !skipTLSVerify {
		var proceed bool

		skipTLSVerify, proceed, err = promptShouldSkipTLSVerify(selectedRegistry.Host)
		if err != nil {
			return err
		}

		if !proceed {
			return fmt.Errorf("could not continue with login")
		}
	}

	client, err := NewApiClient(selectedRegistry.Host, skipTLSVerify)
	if err != nil {
		return err
	}

	sources, _, err := client.SourcesApi.ListSources(context.Background()).Execute()
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		return errors.New("Zero sources have been configured.")
	}

	var selectedSource *api.Source

source:
	switch {
	case len(sources) == 0:
		return errors.New("Zero sources have been configured.")
	case len(sources) == 1:
		selectedSource = &sources[0]
	default:
		// Use the current source ID if it's valid to avoid prompting the user
		if selectedRegistry.SourceID != "" && options.Current {
			for i, source := range sources {
				if source.Id == selectedRegistry.SourceID {
					selectedSource = &sources[i]
					break source
				}
			}
		}

		selectedSource, err = promptSelectSource(sources)
		if errors.Is(err, terminal.InterruptErr) {
			return nil
		}

		if err != nil {
			return err
		}
	}

	var loginReq api.LoginRequest

	switch {
	case selectedSource.Okta != nil:
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

		authorizeURL := "https://" + selectedSource.Okta.Domain + "/oauth2/v1/authorize?redirect_uri=" + "http://localhost:8301&client_id=" + selectedSource.Okta.ClientId + "&response_type=code&scope=openid+email&nonce=" + nonce + "&state=" + state

		fmt.Fprintf(os.Stderr, "%s Logging in with %s...\n", blue("✓"), termenv.String("Okta").Bold().String())

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
			return errors.New("received state is not the same as sent state")
		}

		loginReq.Okta = &api.LoginRequestOkta{
			Domain: selectedSource.Okta.Domain,
			Code:   code,
		}
	default:
		return errors.New("invalid source selected")
	}

	loginRes, _, err := client.AuthApi.Login(context.Background()).Body(loginReq).Execute()
	if err != nil {
		return err
	}

	for i := range loadedCfg.Registries {
		loadedCfg.Registries[i].Current = false
	}

	selectedRegistry.Name = loginRes.Name
	selectedRegistry.Token = loginRes.Token
	selectedRegistry.SkipTLSVerify = skipTLSVerify
	selectedRegistry.SourceID = selectedSource.Id
	selectedRegistry.Current = true

	err = writeConfig(loadedCfg)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "%s Logged in as %s\n", blue("✓"), termenv.String(loginRes.Name).Bold().String())

	client, err = NewApiClient(selectedRegistry.Host, skipTLSVerify)
	if err != nil {
		return err
	}

	users, _, err := client.UsersApi.ListUsers(NewApiContext(loginRes.Token)).Email(loginRes.Name).Execute()
	if err != nil {
		return err
	}

	if len(users) < 1 {
		return fmt.Errorf("User \"%s\" not found", loginRes.Name)
	}

	if len(users) > 1 {
		return fmt.Errorf("Found multiple users \"%s\"", loginRes.Name)
	}

	err = updateKubeconfig(users[0])
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if len(users[0].Roles) > 0 {
		kubeConfigFilename := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ConfigAccess().GetDefaultFilename()
		fmt.Fprintf(os.Stderr, "%s Updated %s\n", blue("✓"), termenv.String(strings.ReplaceAll(kubeConfigFilename, homeDir, "~")).Bold().String())
	}

	context, err := switchToFirstInfraContext()
	if err != nil {
		return err
	}

	if context != "" {
		fmt.Fprintf(os.Stderr, "%s Current Kubernetes context is now %s\n", blue("✓"), termenv.String(context).Bold().String())
	}

	return nil
}

func promptSelectRegistry(registries []ClientRegistryConfig) *ClientRegistryConfig {
	options := []string{}
	for _, reg := range registries {
		options = append(options, reg.Host)
	}

	option := 0
	prompt := &survey.Select{
		Message: "Choose a registry:",
		Options: options,
	}

	err := survey.AskOne(prompt, &option, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = blue("?")
	}))
	if err != nil {
		return nil
	}

	return &registries[option]
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
		if !errors.As(err, &x509.UnknownAuthorityError{}) && !errors.As(err, &x509.HostnameError{}) {
			return false, false, err
		}

		proceed := false

		fmt.Fprintf(os.Stderr, "Could not verify certificate for host %q: %s\n", host, err)

		prompt := &survey.Confirm{
			Message: "Are you sure you want to continue?",
		}

		err := survey.AskOne(prompt, &proceed, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithIcons(func(icons *survey.IconSet) {
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

func promptSelectSource(sources []api.Source) (*api.Source, error) {
	if sources == nil {
		return nil, errors.New("sources cannot be nil")
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

	err := survey.AskOne(prompt, &option, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = blue("?")
	}))
	if errors.Is(err, terminal.InterruptErr) {
		return nil, err
	}

	return &sources[option], nil
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
		for _, c := range kubeConfig.Contexts {
			if !strings.HasPrefix(c.Cluster, "infra:") {
				continue
			}

			// prefer a context with "default" or no namespace
			if c.Namespace == "" || c.Namespace == "default" {
				resultContext = c.Cluster
				break
			}

			resultContext = c.Cluster
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
