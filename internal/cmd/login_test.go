package cmd

import (
	"context"
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
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/uid"
)

func TestLoginCmdSignup(t *testing.T) {
	dir := setupEnv(t)

	opts := defaultServerOptions(dir)
	opts.Addr = server.ListenerOptions{HTTPS: "127.0.0.1:0", HTTP: "127.0.0.1:0"}

	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	setupCertManager(t, opts.TLSCache, srv.Addrs.HTTPS.String())
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
		exp.Send(t, "I do not trust this certificate\n")

		assert.ErrorIs(t, g.Wait(), terminal.InterruptErr)

		// Check we haven't persisted any certificates
		cfg, err := readConfig()
		assert.NilError(t, err)
		assert.Equal(t, len(cfg.Hosts), 0)
	})

	runStep(t, "first login prompts for setup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		console := newConsole(t)
		ctx = PatchCLIWithPTY(ctx, console.Tty())

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			// TODO: remove --skip-tls-verify
			return Run(ctx, "login", srv.Addrs.HTTPS.String(), "--skip-tls-verify")
		})

		exp := expector{console: console}
		exp.ExpectString(t, "Email:")
		exp.Send(t, "admin@example.com\n")
		exp.ExpectString(t, "Password")
		exp.Send(t, "password\n")
		exp.ExpectString(t, "Confirm")
		exp.Send(t, "password\n")
		exp.ExpectString(t, "Logged in as")

		assert.NilError(t, g.Wait())
	})
}

func TestLoginCmd(t *testing.T) {
	dir := setupEnv(t)

	opts := defaultServerOptions(dir)
	opts.Addr = server.ListenerOptions{HTTPS: "127.0.0.1:0", HTTP: "127.0.0.1:0"}
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

	setupCertManager(t, opts.TLSCache, srv.Addrs.HTTPS.String())
	go func() {
		assert.Check(t, srv.Run(ctx))
	}()

	runStep(t, "login without background agent", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		console := newConsole(t)
		ctx = PatchCLIWithPTY(ctx, console.Tty())

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			// TODO: remove --skip-tls-verify
			return Run(ctx, "login", srv.Addrs.HTTPS.String(), "--skip-tls-verify", "--no-agent", "--key", adminAccessKey)
		})

		exp := expector{console: console}
		exp.ExpectString(t, "Logged in as")

		assert.NilError(t, g.Wait())

		_, err := readStoredAgentProcessID()
		assert.ErrorContains(t, err, "no such file or directory")
	})

	runStep(t, "login updated infra config", func(t *testing.T) {
		cfg, err := readConfig()
		assert.NilError(t, err)

		expected := []ClientHostConfig{
			{
				Name:          "admin@example.com",
				AccessKey:     adminAccessKey,
				PolymorphicID: "any-id",
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

	runStep(t, "trust server certificate", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		// Create an access key for login
		cfg, err := readConfig()
		assert.NilError(t, err)
		assert.Equal(t, len(cfg.Hosts), 1)
		userID, _ := cfg.Hosts[0].PolymorphicID.ID()
		client, err := defaultAPIClient()
		assert.NilError(t, err)

		resp, err := client.CreateAccessKey(&api.CreateAccessKeyRequest{
			UserID:            userID,
			TTL:               api.Duration(opts.SessionDuration + time.Minute),
			ExtensionDeadline: api.Duration(time.Minute),
		})
		assert.NilError(t, err)
		accessKey := resp.AccessKey

		// Erase local data for previous login session
		assert.NilError(t, writeConfig(&ClientConfig{}))

		console := newConsole(t)
		ctx = PatchCLIWithPTY(ctx, console.Tty())

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			// TODO: why isn't this working without --non-interactive=false? the other test works
			return Run(ctx, "login", "--non-interactive=false", "--key", accessKey, srv.Addrs.HTTPS.String())
		})
		exp := expector{console: console}
		exp.ExpectString(t, "verify the certificate can be trusted")
		exp.Send(t, "Trust and save the certificate\n")

		assert.NilError(t, g.Wait())

		cert, err := os.ReadFile("testdata/pki/localhost.crt")
		assert.NilError(t, err)

		// Check the client config
		cfg, err = readConfig()
		assert.NilError(t, err)
		expected := []ClientHostConfig{
			{
				Name:               "admin@example.com",
				AccessKey:          "any-access-key",
				PolymorphicID:      "any-id",
				Host:               srv.Addrs.HTTPS.String(),
				Expires:            api.Time(time.Now().UTC().Add(opts.SessionDuration)),
				Current:            true,
				TrustedCertificate: cert,
			},
		}
		// TODO: where is the extra entry coming from?
		assert.DeepEqual(t, cfg.Hosts, expected, cmpClientHostConfig)
	})
}

var cmpClientHostConfig = cmp.Options{
	cmp.FilterPath(
		opt.PathField(ClientHostConfig{}, "AccessKey"),
		cmpStringNotZero),
	cmp.FilterPath(
		opt.PathField(ClientHostConfig{}, "PolymorphicID"),
		cmpPolymorphicIDNotZero),
	cmp.FilterPath(
		opt.PathField(ClientHostConfig{}, "Expires"),
		cmpApiTimeWithThreshold(5*time.Second)),
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

var cmpPolymorphicIDNotZero = cmp.Comparer(func(x, y uid.PolymorphicID) bool {
	return x != "" && y != ""
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

	timeout := time.Second
	if os.Getenv("CI") != "" {
		// CI takes much longer than local dev, use a much longer timeout
		timeout = 20 * time.Second
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

// setupCertManager copies the static TLS cert and key into the cache that will
// be used by the server. This allows the server to skip generating a private key
// for both the CA and server certificate, which takes multiple seconds.
func setupCertManager(t *testing.T, dir string, serverName string) {
	t.Helper()
	ctx := context.Background()
	cache := autocert.DirCache(dir)

	key, err := os.ReadFile("testdata/pki/localhost.key")
	assert.NilError(t, err)
	err = cache.Put(ctx, serverName+".key", key)
	assert.NilError(t, err)

	cert, err := os.ReadFile("testdata/pki/localhost.crt")
	assert.NilError(t, err)
	err = cache.Put(ctx, serverName+".crt", cert)
	assert.NilError(t, err)
}
