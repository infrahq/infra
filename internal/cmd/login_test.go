package cmd

import (
	"context"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/creack/pty"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hinshun/vt10x"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"
	"gotest.tools/v3/golden"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/race"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/uid"
)

var anyUID = uid.ID(99)

func TestLoginCmd_Options(t *testing.T) {
	dir := setupEnv(t)

	opts := defaultServerOptions(dir)
	setupServerTLSOptions(t, &opts)
	adminAccessKey := "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb"
	opts.Config.Users = []server.User{
		{
			Name:      "admin@example.com",
			AccessKey: adminAccessKey,
		},
	}
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		assert.Check(t, srv.Run(ctx))
	}()

	runStep(t, "login without background agent", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		// TODO: remove --skip-tls-verify
		err := Run(ctx, "login", srv.Addrs.HTTPS.String(), "--skip-tls-verify", "--no-agent", "--key", adminAccessKey)
		assert.NilError(t, err)

		_, err = readStoredAgentProcessID()
		assert.ErrorContains(t, err, "no such file or directory")
	})

	runStep(t, "login updated infra config", func(t *testing.T) {
		cfg, err := readConfig()
		assert.NilError(t, err)
		expected := []ClientHostConfig{
			{
				Name:          "admin@example.com",
				AccessKey:     adminAccessKey,
				UserID:        anyUID,
				Host:          srv.Addrs.HTTPS.String(),
				SkipTLSVerify: true,
				Expires:       api.Time(time.Now().UTC().Add(opts.SessionDuration)),
				Current:       true,
			},
		}
		assert.DeepEqual(t, cfg.Hosts, expected, cmpClientHostConfig)
	})

	runStep(t, "login updated kube config", func(t *testing.T) {
		kubeCfg, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		// config is empty because there are no grants yet
		expected := clientcmdapi.Config{}
		assert.DeepEqual(t, expected, kubeCfg, cmpopts.EquateEmpty())
	})
}

var cmpClientHostConfig = cmp.Options{
	cmp.FilterPath(
		opt.PathField(ClientHostConfig{}, "AccessKey"),
		cmpStringNotZero),
	cmp.FilterPath(
		opt.PathField(ClientHostConfig{}, "UserID"),
		cmpUserIDNotZero),
	cmp.FilterPath(
		opt.PathField(ClientHostConfig{}, "Expires"),
		cmpApiTimeWithThreshold(20*time.Second)),
}

func cmpApiTimeWithThreshold(threshold time.Duration) cmp.Option {
	return cmp.Comparer(func(xa, ya api.Time) bool {
		x, y := time.Time(xa), time.Time(ya)
		if x.IsZero() || y.IsZero() {
			return false
		}
		delta := x.Sub(y)
		return delta <= threshold && delta >= -threshold
	})
}

var cmpStringNotZero = cmp.Comparer(func(x, y string) bool {
	return x != "" && y != ""
})

var cmpUserIDNotZero = cmp.Comparer(func(x, y uid.ID) bool {
	return x > 0 && y > 0
})

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

func newConsole(t *testing.T) *expect.Console {
	t.Helper()

	pseudoTY, tty, err := pty.Open()
	assert.NilError(t, err, "failed to open pseudo tty")

	timeout := 10 * time.Second
	if os.Getenv("CI") != "" || race.Enabled {
		// CI and -race take much longer than regular runs, use a much longer timeout
		timeout = 30 * time.Second
	}

	term := vt10x.New(vt10x.WithWriter(tty))
	console, err := expect.NewConsole(
		// expect.WithLogger(log.New(os.Stderr, "", 0)),
		expect.WithDefaultTimeout(timeout),
		expect.WithStdout(os.Stdout),
		expect.WithStdin(pseudoTY),
		expect.WithStdout(term),
		expect.WithCloser(pseudoTY, tty))
	assert.NilError(t, err)
	t.Cleanup(func() {
		// make sure stdout has newlines to prevent test2json parse failures,
		// and leaking control sequences to stdout.
		t.Log("\n")
		console.Close()
	})
	return console
}

type expector struct {
	console *expect.Console
}

func (e *expector) ExpectString(t *testing.T, v string) {
	t.Helper()
	_, err := e.console.ExpectString(v)
	assert.NilError(t, err, "expected string: %v", v)
}

func (e *expector) Send(t *testing.T, v string) {
	t.Helper()
	_, err := e.console.Send(v)
	assert.NilError(t, err, "sending string: %v", v)
}

// setupEnv sets the environment variable that the CLI expects
func setupEnv(t *testing.T) string {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(dir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	return dir
}

func setupServerTLSOptions(t *testing.T, opts *server.Options) {
	t.Helper()

	opts.Config.OrganizationName = "CLI test"
	opts.Config.OrganizationDomain = "cli-test"

	opts.Addr = server.ListenerOptions{HTTPS: "127.0.0.1:0", HTTP: "127.0.0.1:0"}

	key, err := os.ReadFile("testdata/pki/localhost.key")
	assert.NilError(t, err)
	opts.TLS.PrivateKey = string(key)

	cert, err := os.ReadFile("testdata/pki/localhost.crt")
	assert.NilError(t, err)
	opts.TLS.Certificate = types.StringOrFile(cert)
}

func TestLoginCmd_TLSVerify(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(dir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	opts := defaultServerOptions(dir)
	setupServerTLSOptions(t, &opts)
	accessKey := "0000000001.adminadminadminadmin1234"
	opts.Users = []server.User{
		{Name: "admin@example.com", AccessKey: accessKey},
	}
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		assert.Check(t, srv.Run(ctx))
	}()

	runStep(t, "reject server certificate", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		console := newConsole(t)
		ctx = PatchCLIWithPTY(ctx, console.Tty())

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			// TODO: why isn't this working without --non-interactive=false? the other test works
			return Run(ctx, "login", "--non-interactive=false", srv.Addrs.HTTPS.String())
		})
		exp := expector{console: console}
		exp.ExpectString(t, "verify the certificate can be trusted")
		exp.Send(t, "NO\n")

		assert.ErrorIs(t, g.Wait(), terminal.InterruptErr)

		// Check we haven't persisted any certificates
		cfg, err := readConfig()
		assert.NilError(t, err)
		assert.Equal(t, len(cfg.Hosts), 0)
	})

	runStep(t, "trust server certificate", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		console := newConsole(t)
		ctx = PatchCLIWithPTY(ctx, console.Tty())

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			// TODO: why isn't this working without --non-interactive=false? the other test works
			return Run(ctx, "login", "--non-interactive=false", "--key", accessKey, srv.Addrs.HTTPS.String())
		})
		exp := expector{console: console}
		exp.ExpectString(t, "verify the certificate can be trusted")
		exp.Send(t, "TRUST\n")

		assert.NilError(t, g.Wait())

		cert, err := os.ReadFile("testdata/pki/localhost.crt")
		assert.NilError(t, err)

		// Check the client config
		cfg, err := readConfig()
		assert.NilError(t, err)
		expected := []ClientHostConfig{
			{
				Name:               "admin@example.com",
				AccessKey:          "any-access-key",
				UserID:             anyUID,
				Host:               srv.Addrs.HTTPS.String(),
				Expires:            api.Time(time.Now().UTC().Add(opts.SessionDuration)),
				Current:            true,
				TrustedCertificate: string(cert),
			},
		}
		assert.DeepEqual(t, cfg.Hosts, expected, cmpClientHostConfig)
	})

	runStep(t, "next login should still trust the server", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		err := Run(ctx, "logout")
		assert.NilError(t, err)

		err = Run(ctx, "login", "--key", accessKey, srv.Addrs.HTTPS.String())
		assert.NilError(t, err)

		cert, err := os.ReadFile("testdata/pki/localhost.crt")
		assert.NilError(t, err)

		// Check the client config
		cfg, err := readConfig()
		assert.NilError(t, err)
		expected := []ClientHostConfig{
			{
				Name:               "admin@example.com",
				AccessKey:          "any-access-key",
				UserID:             anyUID,
				Host:               srv.Addrs.HTTPS.String(),
				Expires:            api.Time(time.Now().UTC().Add(opts.SessionDuration)),
				Current:            true,
				TrustedCertificate: string(cert),
			},
		}
		assert.DeepEqual(t, cfg.Hosts, expected, cmpClientHostConfig)
	})

	t.Run("login with trusted cert", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		err := Run(ctx, "logout", "--clear")
		assert.NilError(t, err)

		err = Run(ctx, "login",
			"--tls-trusted-cert", "testdata/pki/localhost.crt",
			"--key", accessKey,
			srv.Addrs.HTTPS.String())
		assert.NilError(t, err)

		cert, err := os.ReadFile("testdata/pki/localhost.crt")
		assert.NilError(t, err)

		// Check the client config
		cfg, err := readConfig()
		assert.NilError(t, err)
		expected := []ClientHostConfig{
			{
				Name:               "admin@example.com",
				AccessKey:          "any-access-key",
				UserID:             anyUID,
				Host:               srv.Addrs.HTTPS.String(),
				Expires:            api.Time(time.Now().UTC().Add(opts.SessionDuration)),
				Current:            true,
				TrustedCertificate: string(cert),
			},
		}
		assert.DeepEqual(t, cfg.Hosts, expected, cmpClientHostConfig)
	})

	t.Run("login with trusted fingerprint", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)
		t.Setenv("INFRA_ACCESS_KEY", accessKey)
		t.Setenv("INFRA_SERVER", srv.Addrs.HTTPS.String())

		err := Run(ctx, "logout", "--clear")
		assert.NilError(t, err)

		cert, err := os.ReadFile("testdata/pki/localhost.crt")
		assert.NilError(t, err)

		block, _ := pem.Decode(cert)
		fingerprint := certs.Fingerprint(block.Bytes)

		err = Run(ctx, "login",
			"--tls-trusted-fingerprint", fingerprint)
		assert.NilError(t, err)

		// Check the client config
		cfg, err := readConfig()
		assert.NilError(t, err)
		expected := []ClientHostConfig{
			{
				Name:               "admin@example.com",
				AccessKey:          "any-access-key",
				UserID:             anyUID,
				Host:               srv.Addrs.HTTPS.String(),
				Expires:            api.Time(time.Now().UTC().Add(opts.SessionDuration)),
				Current:            true,
				TrustedCertificate: string(cert),
			},
		}
		assert.DeepEqual(t, cfg.Hosts, expected, cmpClientHostConfig)
	})

	t.Run("login with wrong fingerprint", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		err := Run(ctx, "logout", "--clear")
		assert.NilError(t, err)

		ctx, bufs := PatchCLI(ctx)

		err = Run(ctx, "login",
			"--tls-trusted-fingerprint", "BA::D0::FF",
			"--key", accessKey,
			srv.Addrs.HTTPS.String())
		assert.ErrorContains(t, err, "authenticity of the server could not be verified")

		// Check the client config is empty
		cfg, err := readConfig()
		assert.NilError(t, err)
		expected := &ClientConfig{ClientConfigVersion: clientConfigVersion}
		assert.DeepEqual(t, cfg, expected, cmpopts.EquateEmpty())

		golden.Assert(t, bufs.Stderr.String(), t.Name())
	})
}

func TestAuthURLForProvider(t *testing.T) {
	expectedOktaAuthURL := "https://okta.example.com/oauth2/v1/authorize?client_id=001&redirect_uri=http%3A%2F%2Flocalhost%3A8301&response_type=code&scope=email+openid&state=state"
	okta := api.Provider{
		AuthURL:  "https://okta.example.com/oauth2/v1/authorize",
		ClientID: "001",
		Kind:     "okta",
		Scopes: []string{
			"email",
			"openid",
		},
	}
	url, err := authURLForProvider(okta, "state")
	assert.NilError(t, err)
	assert.Equal(t, url, expectedOktaAuthURL)

	expectedAzureAuthURL := "https://login.microsoftonline.com/0/oauth2/v2.0/authorize?client_id=001&redirect_uri=http%3A%2F%2Flocalhost%3A8301&response_type=code&scope=email+openid&state=state"
	azure := api.Provider{
		AuthURL:  "https://login.microsoftonline.com/0/oauth2/v2.0/authorize",
		ClientID: "001",
		Kind:     "azure",
		Scopes: []string{
			"email",
			"openid",
		},
	}
	url, err = authURLForProvider(azure, "state")
	assert.NilError(t, err)
	assert.Equal(t, url, expectedAzureAuthURL)

	expectedGoogleAuthURL := "https://accounts.google.com/o/oauth2/v2/auth?access_type=offline&client_id=001&prompt=consent&redirect_uri=http%3A%2F%2Flocalhost%3A8301&response_type=code&scope=email+openid&state=state"
	google := api.Provider{
		AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
		ClientID: "001",
		Kind:     "google",
		Scopes: []string{
			"email",
			"openid",
		},
	}
	url, err = authURLForProvider(google, "state")
	assert.NilError(t, err)
	assert.Equal(t, url, expectedGoogleAuthURL)

	// test that the client resolve the auth URL when the server does not send it
	// this test does an external call to example.okta.com, if it fails check your network connection
	expectedResolvedAuthURL := "https://example.okta.com/oauth2/v1/authorize?client_id=001&redirect_uri=http%3A%2F%2Flocalhost%3A8301&response_type=code&scope=openid+email+offline_access+groups&state=state"
	oldProvider := api.Provider{
		// no AuthURL set
		URL:      "example.okta.com",
		ClientID: "001",
		Kind:     "okta",
		Scopes: []string{
			"email",
			"openid",
		},
	}
	url, err = authURLForProvider(oldProvider, "state")
	assert.NilError(t, err)
	assert.Equal(t, url, expectedResolvedAuthURL)
}
