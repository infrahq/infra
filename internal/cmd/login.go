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
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/muesli/termenv"
	"golang.org/x/term"
	"k8s.io/client-go/tools/clientcmd"
)

type LoginOptions struct {
	Current bool          `mapstructure:"current"`
	Timeout time.Duration `mapstructure:"timeout"`
	*internal.GlobalOptions
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

host:
	switch {
	case options.Host == "":
		if options.Current {
			for i := range loadedCfg.Hosts {
				if loadedCfg.Hosts[i].Current {
					selectedHost = &loadedCfg.Hosts[i]
					break host
				}
			}
		}

		// TODO (https://github.com/infrahq/infra/issues/496): prompt user instead of assuming the first hostname
		// since they may not know where they are logging into
		if len(loadedCfg.Hosts) == 1 {
			selectedHost = &loadedCfg.Hosts[0]
			break
		}

		selectedHost = promptSelectHost(loadedCfg.Hosts)
	default:
		for i := range loadedCfg.Hosts {
			if loadedCfg.Hosts[i].Host == options.Host {
				selectedHost = &loadedCfg.Hosts[i]
				break host
			}
		}

		loadedCfg.Hosts = append(loadedCfg.Hosts, ClientHostConfig{
			Host:    options.Host,
			Current: true,
		})
		selectedHost = &loadedCfg.Hosts[len(loadedCfg.Hosts)-1]
	}

	if selectedHost == nil {
		return errors.New("host endpoint is required to login")
	}

	fmt.Fprintf(os.Stderr, "%s Logging in to %s\n", blue("✓"), termenv.String(selectedHost.Host).Bold().String())

	skipTLSVerify := selectedHost.SkipTLSVerify
	if !skipTLSVerify {
		var proceed bool

		skipTLSVerify, proceed, err = promptShouldSkipTLSVerify(selectedHost.Host)
		if err != nil {
			return err
		}

		if !proceed {
			return fmt.Errorf("user declined login")
		}
	}

	client, err := NewAPIClient(selectedHost.Host, skipTLSVerify)
	if err != nil {
		return err
	}

	sources, res, err := client.SourcesAPI.ListSources(context.Background()).Execute()
	if err != nil {
		return errWithResponseContext(err, res)
	}

	var selectedSource *api.Source

source:
	switch {
	case len(sources) == 0:
		return errors.New("no identity providers have been configured")
	case len(sources) == 1:
		selectedSource = &sources[0]
	default:
		// Use the current source ID if it's valid to avoid prompting the user
		if selectedHost.SourceID != "" && options.Current {
			for i, source := range sources {
				if source.Id == selectedHost.SourceID {
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

	loginRes, res, err := client.AuthAPI.Login(context.Background()).Body(loginReq).Execute()
	if err != nil {
		return errWithResponseContext(err, res)
	}

	for i := range loadedCfg.Hosts {
		loadedCfg.Hosts[i].Current = false
	}

	selectedHost.Name = loginRes.Name
	selectedHost.Token = loginRes.Token
	selectedHost.SkipTLSVerify = skipTLSVerify
	selectedHost.SourceID = selectedSource.Id
	selectedHost.Current = true

	err = writeConfig(loadedCfg)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "%s Logged in as %s\n", blue("✓"), termenv.String(loginRes.Name).Bold().String())

	client, err = NewAPIClient(selectedHost.Host, skipTLSVerify)
	if err != nil {
		return err
	}

	users, res, err := client.UsersAPI.ListUsers(NewAPIContext(loginRes.Token)).Email(loginRes.Name).Execute()
	if err != nil {
		return errWithResponseContext(err, res)
	}

	if len(users) < 1 {
		return fmt.Errorf("user \"%s\" not found", loginRes.Name)
	}

	if len(users) > 1 {
		return fmt.Errorf("found multiple users \"%s\"", loginRes.Name)
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

func promptSelectHost(hosts []ClientHostConfig) *ClientHostConfig {
	options := []string{}
	for _, reg := range hosts {
		options = append(options, reg.Host)
	}

	option := 0
	prompt := &survey.Select{
		Message: "Choose Infra account:",
		Options: options,
	}

	err := survey.AskOne(prompt, &option, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = blue("?")
	}))
	if err != nil {
		return nil
	}

	return &hosts[option]
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
