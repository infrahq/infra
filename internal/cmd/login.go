package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/goware/urlx"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
)

type loginCmdOptions struct {
	Server             string
	AccessKey          string
	Provider           string
	SkipTLSVerify      bool
	TrustedCertificate string
	TrustedFingerprint string
	NonInteractive     bool
	NoAgent            bool
}

type loginMethod int8

const (
	localLogin loginMethod = iota
	oidcLogin
)

const cliLoginRedirectURL = "http://localhost:8301"

func newLoginCmd(cli *CLI) *cobra.Command {
	var options loginCmdOptions

	cmd := &cobra.Command{
		Use:   "login [SERVER]",
		Short: "Login to Infra",
		Long:  "Login to Infra and start a background agent to keep local configuration up-to-date",
		Example: `# By default, login will prompt for all required information.
$ infra login

# Login to a specific server
$ infra login infraexampleserver.com

# Login with a specific identity provider
$ infra login --provider okta

# Login with an access key
$ export INFRA_ACCESS_KEY=1M4CWy9wF5.fAKeKEy5sMLH9ZZzAur0ZIjy
$ infra login`,
		Args:  MaxArgs(1),
		Group: "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server, ok := os.LookupEnv("INFRA_SERVER"); ok {
				options.Server = server
			}

			if len(args) == 1 {
				options.Server = args[0]
			}

			return login(cli, options)
		},
	}

	cmd.Flags().StringVar(&options.AccessKey, "key", "", "Login with an access key")
	cmd.Flags().StringVar(&options.Provider, "provider", "", "Login with an identity provider")
	cmd.Flags().BoolVar(&options.SkipTLSVerify, "skip-tls-verify", false, "Skip verifying server TLS certificates")
	cmd.Flags().Var((*types.StringOrFile)(&options.TrustedCertificate), "tls-trusted-cert", "TLS certificate or CA used by the server")
	cmd.Flags().StringVar(&options.TrustedFingerprint, "tls-trusted-fingerprint", "", "SHA256 fingerprint of the server TLS certificate")
	cmd.Flags().BoolVar(&options.NoAgent, "no-agent", false, "Skip starting the Infra agent in the background")
	addNonInteractiveFlag(cmd.Flags(), &options.NonInteractive)
	return cmd
}

func login(cli *CLI, options loginCmdOptions) error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	if options.Server == "" {
		if options.NonInteractive {
			return Error{Message: "Non-interactive login requires the [SERVER] argument or environment variable INFRA_SERVER to be set"}
		}

		options.Server, err = promptServer(cli, config)
		if err != nil {
			return err
		}
	}

	if len(options.TrustedCertificate) == 0 {
		// Attempt to find a previously trusted certificate
		for _, hc := range config.Hosts {
			if equalHosts(hc.Host, options.Server) {
				options.TrustedCertificate = hc.TrustedCertificate
			}
		}
	}

	lc, err := newLoginClient(cli, options)
	if err != nil {
		return err
	}

	loginReq := &api.LoginRequest{}

	// if signup is required, use it to create an admin account
	// and use those credentials for subsequent requests
	logging.Debugf("call server: check signup enabled")
	signupEnabled, err := lc.APIClient.SignupEnabled()
	if err != nil {
		return err
	}

	if signupEnabled.Enabled {
		loginReq.PasswordCredentials, err = runSignupForLogin(cli, lc.APIClient)
		if err != nil {
			return err
		}

		return loginToInfra(cli, lc, loginReq, options.NoAgent)
	}

	if options.AccessKey == "" {
		options.AccessKey = os.Getenv("INFRA_ACCESS_KEY")
	}

	switch {
	case options.AccessKey != "":
		loginReq.AccessKey = options.AccessKey
	case options.Provider != "":
		if options.NonInteractive {
			return Error{Message: "Non-interactive login only supports access keys, set the INFRA_ACCESS_KEY environment variable and try again"}
		}
		loginReq.OIDC, err = loginToProviderN(lc.APIClient, options.Provider)
		if err != nil {
			return err
		}
	default:
		if options.NonInteractive {
			return Error{Message: "Non-interactive login only supports access keys, set the INFRA_ACCESS_KEY environment variable and try again"}
		}
		loginMethod, provider, err := promptLoginOptions(cli, lc.APIClient)
		if err != nil {
			return err
		}

		switch loginMethod {
		case localLogin:
			loginReq.PasswordCredentials, err = promptLocalLogin(cli)
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

	return loginToInfra(cli, lc, loginReq, options.NoAgent)
}

func equalHosts(x, y string) bool {
	if x == y {
		return true
	}
	if strings.TrimPrefix(x, "https://") == strings.TrimPrefix(y, "https://") {
		return true
	}
	return false
}

func loginToInfra(cli *CLI, lc loginClient, loginReq *api.LoginRequest, noAgent bool) error {
	loginRes, err := lc.APIClient.Login(loginReq)
	if err != nil {
		logging.Debugf("login: %s", err)
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
	// Update the API client with the new access key from login
	lc.APIClient.AccessKey = loginRes.AccessKey

	if loginRes.PasswordUpdateRequired {
		fmt.Fprintf(cli.Stderr, "  Your password has expired. Please update your password (min. length 8).\n")

		password, err := promptSetPassword(cli, loginReq.PasswordCredentials.Password)
		if err != nil {
			return err
		}

		logging.Debugf("call server: update user %s", loginRes.UserID)
		if _, err := lc.APIClient.UpdateUser(&api.UpdateUserRequest{ID: loginRes.UserID, Password: password}); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "  Updated password.\n")
	}

	if err := updateInfraConfig(lc, loginReq, loginRes); err != nil {
		return err
	}

	if err := updateKubeConfig(lc.APIClient, loginRes.UserID); err != nil {
		return err
	}

	backgroundAgentRunning, err := configAgentRunning()
	if err != nil {
		// do not block login, just proceed, potentially without the agent
		logging.Errorf("unable to check background agent: %v", err)
	}

	if !backgroundAgentRunning && !noAgent {
		// the agent is started in a separate command so that it continues after the login command has finished
		if err := execAgent(); err != nil {
			// user still has a valid session, so do not fail
			logging.Errorf("Unable to start agent, destinations will not be updated automatically: %v", err)
		}
	}

	fmt.Fprintf(cli.Stderr, "  Logged in as %s\n", termenv.String(loginRes.Name).Bold().String())
	return nil
}

// Updates all configs with the current logged in session
func updateInfraConfig(lc loginClient, loginReq *api.LoginRequest, loginRes *api.LoginResponse) error {
	clientHostConfig := ClientHostConfig{
		Current:   true,
		UserID:    loginRes.UserID,
		Name:      loginRes.Name,
		AccessKey: loginRes.AccessKey,
		Expires:   loginRes.Expires,
	}

	t, ok := lc.APIClient.HTTP.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("could not update infra config")
	}
	clientHostConfig.SkipTLSVerify = t.TLSClientConfig.InsecureSkipVerify
	if lc.TrustedCertificate != "" {
		clientHostConfig.TrustedCertificate = lc.TrustedCertificate
	}

	if loginReq.OIDC != nil {
		clientHostConfig.ProviderID = loginReq.OIDC.ProviderID
	}

	u, err := urlx.Parse(lc.APIClient.URL)
	if err != nil {
		return err
	}
	clientHostConfig.Host = u.Host

	if err := saveHostConfig(clientHostConfig); err != nil {
		return err
	}

	return nil
}

func oidcflow(provider *api.Provider) (string, error) {
	// the state makes sure we are getting the correct response for our request
	state, err := generate.CryptoRandom(12, generate.CharsetAlphaNumeric)
	if err != nil {
		return "", err
	}

	authorizeURL := fmt.Sprintf("%s?redirect_uri=http://localhost:8301&client_id=%s&response_type=code&scope=%s&state=%s", provider.AuthURL, provider.ClientID, strings.Join(provider.Scopes, "+"), state)

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
		return "", Error{Message: "Login aborted, provider state did not match the expected state"}
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

	code, err := oidcflow(provider)
	if err != nil {
		return nil, err
	}

	return &api.LoginRequestOIDC{
		ProviderID:  provider.ID,
		RedirectURL: cliLoginRedirectURL,
		Code:        code,
	}, nil
}

func runSignupForLogin(cli *CLI, client *api.Client) (*api.LoginRequestPasswordCredentials, error) {
	fmt.Fprintln(cli.Stderr, "  Welcome to Infra. Set up your admin user:")

	email, err := promptSetEmail(cli)
	if err != nil {
		return nil, err
	}

	password, err := promptSetPassword(cli, "")
	if err != nil {
		return nil, err
	}

	logging.Debugf("call server: signup for user %q", email)
	_, err = client.Signup(&api.SignupRequest{Name: email, Password: password})
	if err != nil {
		return nil, err
	}

	return &api.LoginRequestPasswordCredentials{
		Name:     email,
		Password: password,
	}, nil
}

type loginClient struct {
	APIClient *api.Client
	// TrustedCertificate is a PEM encoded certificate that has been trusted by
	// the user for TLS communication with the server.
	TrustedCertificate string
}

// Only used when logging in or switching to a new session, since user has no credentials. Otherwise, use defaultAPIClient().
func newLoginClient(cli *CLI, options loginCmdOptions) (loginClient, error) {
	cfg := &ClientHostConfig{
		TrustedCertificate: options.TrustedCertificate,
		SkipTLSVerify:      options.SkipTLSVerify,
	}
	c := loginClient{
		APIClient:          apiClient(options.Server, "", httpTransportForHostConfig(cfg)),
		TrustedCertificate: options.TrustedCertificate,
	}
	if options.SkipTLSVerify {
		return c, nil
	}

	// Prompt user only if server fails the TLS verification
	if err := attemptTLSRequest(options); err != nil {
		var uaErr x509.UnknownAuthorityError
		if !errors.As(err, &uaErr) {
			return c, err
		}

		if !fingerprintMatch(cli, options.TrustedFingerprint, uaErr.Cert) {
			if options.NonInteractive {
				if options.TrustedCertificate != "" {
					return c, err
				}
				return c, Error{
					Message: "The authenticity of the server could not be verified. " +
						"Use the --tls-trusted-cert flag to specify a trusted CA, or run " +
						"in interactive mode.",
				}
			}

			if err := promptVerifyTLSCert(cli, uaErr.Cert); err != nil {
				return c, err
			}
		}

		pool, err := x509.SystemCertPool()
		if err != nil {
			return c, err
		}
		pool.AddCert(uaErr.Cert)
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				// set min version to the same as default to make gosec linter happy
				MinVersion: tls.VersionTLS12,
				RootCAs:    pool,
			},
		}
		c.APIClient = apiClient(options.Server, "", transport)
		c.TrustedCertificate = string(certs.PEMEncodeCertificate(uaErr.Cert.Raw))
	}
	return c, nil
}

func fingerprintMatch(cli *CLI, fingerprint string, cert *x509.Certificate) bool {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return false
	}

	if strings.EqualFold(fingerprint, certs.Fingerprint(cert.Raw)) {
		return true
	}

	fmt.Fprintf(cli.Stderr, `
%v TLS fingerprint from server does not match the trusted fingerprint.

Trusted: %v
Server:  %v
`,
		termenv.String("WARNING").Bold().String(),
		fingerprint,
		certs.Fingerprint(cert.Raw))
	return false
}

func attemptTLSRequest(options loginCmdOptions) error {
	reqURL := "https://" + options.Server
	// First attempt with the system cert pool
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	logging.Debugf("call server: test tls for %q", reqURL)
	httpClient := http.Client{Timeout: 60 * time.Second}
	res, err := httpClient.Do(req)
	if err == nil {
		res.Body.Close()
		return nil
	}

	// Second attempt with an empty cert pool. This is necessary because at least
	// on darwin, the error is the wrong type when using the system cert pool.
	// See https://github.com/golang/go/issues/52010.
	req, err = http.NewRequestWithContext(context.TODO(), http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	pool := x509.NewCertPool()
	if options.TrustedCertificate != "" {
		pool.AppendCertsFromPEM([]byte(options.TrustedCertificate))
	}

	httpClient = http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		},
	}
	res, err = httpClient.Do(req)
	urlErr := &url.Error{}
	switch {
	case err == nil:
		res.Body.Close()
		return nil
	case errors.As(err, &urlErr):
		if urlErr.Timeout() {
			return fmt.Errorf("%w: %s", api.ErrTimeout, err)
		}
	}
	return err
}

func promptLocalLogin(cli *CLI) (*api.LoginRequestPasswordCredentials, error) {
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

	if err := survey.Ask(questionPrompt, &credentials, cli.surveyIO); err != nil {
		return &api.LoginRequestPasswordCredentials{}, err
	}

	return &api.LoginRequestPasswordCredentials{
		Name:     credentials.Username,
		Password: credentials.Password,
	}, nil
}

func listProviders(client *api.Client) ([]api.Provider, error) {
	logging.Debugf("call server: list providers")
	providers, err := client.ListProviders("")
	if err != nil {
		return nil, err
	}

	return providers.Items, nil
}

func promptLoginOptions(cli *CLI, client *api.Client) (loginMethod loginMethod, provider *api.Provider, err error) {
	providers, err := listProviders(client)
	if err != nil {
		return 0, nil, err
	}

	var options []string
	for _, p := range providers {
		options = append(options, fmt.Sprintf("%s (%s)", p.Name, p.URL))
	}

	options = append(options, "Login with username and password")

	var i int
	selectPrompt := &survey.Select{
		Message: "Select a login method:",
		Options: options,
	}
	err = survey.AskOne(selectPrompt, &i, cli.surveyIO)
	if errors.Is(err, terminal.InterruptErr) {
		return 0, nil, err
	}

	if i == len(options)-1 {
		return localLogin, nil, nil
	}
	return oidcLogin, &providers[i], nil
}

func promptVerifyTLSCert(cli *CLI, cert *x509.Certificate) error {
	formatTime := func(t time.Time) string {
		return fmt.Sprintf("%v (%v)", HumanTime(t, "none"), t.Format(time.RFC1123))
	}
	title := "Certificate"
	if cert.IsCA {
		title = "Certificate Authority"
	}

	// TODO: improve this message
	// TODO: use color/bold to highlight important parts
	fmt.Fprintf(cli.Stderr, `
The certificate presented by the server is not trusted by your operating system.

%[6]v

Subject: %[1]s
Issuer: %[2]s

Validity
  Not Before: %[3]v
  Not After:  %[4]v

SHA256 Fingerprint
  %[5]v

Compare the SHA256 fingerprint to the one provided by your administrator to
manually verify the certificate can be trusted.

`,
		cert.Subject,
		cert.Issuer,
		formatTime(cert.NotBefore),
		formatTime(cert.NotAfter),
		certs.Fingerprint(cert.Raw),
		title,
	)
	confirmPrompt := &survey.Select{
		Message: "Options:",
		Options: []string{
			"NO",
			"TRUST",
		},
		Description: func(value string, index int) string {
			switch value {
			case "NO":
				return "I do not trust this certificate"
			case "TRUST":
				return "Trust and save the certificate"
			default:
				return ""
			}
		},
	}
	var selection string
	if err := survey.AskOne(confirmPrompt, &selection, cli.surveyIO); err != nil {
		return err
	}
	if selection == "TRUST" {
		return nil
	}
	return terminal.InterruptErr
}

// Returns the host address of the Infra server that user would like to log into
func promptServer(cli *CLI, config *ClientConfig) (string, error) {
	servers := config.Hosts

	if len(servers) == 0 {
		return promptNewServer(cli)
	}

	return promptServerList(cli, servers)
}

func promptNewServer(cli *CLI) (string, error) {
	var server string
	err := survey.AskOne(
		&survey.Input{Message: "Server:"},
		&server,
		cli.surveyIO,
		survey.WithValidator(survey.Required),
	)
	return server, err
}

func promptServerList(cli *CLI, servers []ClientHostConfig) (string, error) {
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
	if err := survey.AskOne(prompt, &i, survey.WithFilter(filter), cli.surveyIO); err != nil {
		return "", err
	}

	if promptOptions[i] == defaultOption {
		return promptNewServer(cli)
	}

	return servers[i].Host, nil
}

func promptSetEmail(cli *CLI) (string, error) {
	var email string
PROMPT:
	if err := survey.AskOne(
		&survey.Input{Message: "Email:"},
		&email,
		cli.surveyIO,
		survey.WithValidator(survey.Required),
	); err != nil {
		return "", err
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		cli.Output("  Please enter a valid email.")
		goto PROMPT
	}

	return email, nil
}
