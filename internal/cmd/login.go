package cmd

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/gofrs/flock"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/muesli/termenv"
	"k8s.io/client-go/tools/clientcmd"
)

type ErrUnauthenticated struct{}

func (e *ErrUnauthenticated) Error() string {
	return "Could not read local credentials. Are you logged in? Use \"infra login\" to login."
}

func login(registry string, useCurrentConfig bool) error {
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
			fmt.Fprintln(os.Stderr, "Failed to unlock login.")
		}
	}()

	if !acquired {
		fmt.Fprintln(os.Stderr, "Another instance is already trying to login.")

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		_, err = lock.TryLockContext(ctx, time.Second*1)
		if err != nil {
			return err
		}

		return nil
	}

	loadedCfg, err := readConfig()
	if err != nil && !errors.Is(err, &ErrUnauthenticated{}) {
		return err
	}

	var selectedRegistry *ClientRegistryConfig

	if loadedCfg != nil {
		if len(registry) == 0 && len(loadedCfg.Registries) == 1 {
			selectedRegistry = &loadedCfg.Registries[0]
		}

		if len(registry) == 0 && len(loadedCfg.Registries) > 1 && useCurrentConfig {
			for i := range loadedCfg.Registries {
				if loadedCfg.Registries[i].Current {
					selectedRegistry = &loadedCfg.Registries[i]
					break
				}
			}
		}

		if len(registry) == 0 && len(loadedCfg.Registries) > 1 && !useCurrentConfig {
			selectedRegistry = promptSelectRegistry(loadedCfg.Registries)
		}

		if len(registry) > 0 && len(loadedCfg.Registries) > 0 {
			for i := range loadedCfg.Registries {
				if loadedCfg.Registries[i].Host == registry {
					selectedRegistry = &loadedCfg.Registries[i]
					break
				}
			}
		}
	}

	if loadedCfg == nil {
		loadedCfg = NewClientConfig()
	}

	if len(registry) > 0 && selectedRegistry == nil {
		// user is specifying a new registry
		loadedCfg.Registries = append(loadedCfg.Registries, ClientRegistryConfig{
			Host:    registry,
			Current: true,
		})
		selectedRegistry = &loadedCfg.Registries[len(loadedCfg.Registries)-1]
	}

	if selectedRegistry == nil {
		// at this point they have not specified a registry and have none to choose from.
		return errors.New("A registry endpoint is required to continue with login.")
	}

	fmt.Fprintf(os.Stderr, "%s Logging in to %s\n", blue("✓"), termenv.String(selectedRegistry.Host).Bold().String())

	skipTLSVerify, proceed, err := promptShouldSkipTLSVerify(selectedRegistry.Host, selectedRegistry.SkipTLSVerify)
	if err != nil {
		return err
	}

	if !proceed {
		return fmt.Errorf("Could not continue with login.")
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

	source, err := promptSelectSource(sources, selectedRegistry.SourceID)

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

		fmt.Fprintf(os.Stderr, "%s Logging in with %s...\n", blue("✓"), termenv.String("Okta").Bold().String())

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

	for i := range loadedCfg.Registries {
		loadedCfg.Registries[i].Current = false
	}

	selectedRegistry.Name = loginRes.Name
	selectedRegistry.Token = loginRes.Token
	selectedRegistry.SkipTLSVerify = skipTLSVerify
	selectedRegistry.SourceID = source.Id
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
		fmt.Fprintf(os.Stderr, "%s Updated %s\n", blue("✓"), termenv.String(strings.ReplaceAll(kubeConfigPath, homeDir, "~")).Bold().String())
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

	err := survey.AskOne(prompt, &option, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = blue("?")
	}))
	if err != nil {
		return nil
	}

	return &registries[option]
}

func promptShouldSkipTLSVerify(host string, skipTLSVerify bool) (shouldSkipTLSVerify bool, proceed bool, err error) {
	if skipTLSVerify {
		return true, true, nil
	}

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

		fmt.Fprintf(os.Stderr, "Could not verify certificate for host %s\n", termenv.String(host).Bold())

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

	if len(sources) == 1 {
		return &sources[0], nil
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
