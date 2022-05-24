package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestLoginCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(dir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	opts := defaultServerOptions(dir)
	opts.Addr = server.ListenerOptions{HTTPS: "127.0.0.1:0", HTTP: "127.0.0.1:0"}
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	setupCertManager(t, opts.TLSCache, srv.Addrs.HTTPS.String())
	go func() {
		err := srv.Run(ctx)
		assert.Check(t, err)
	}()

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
		exp.ExpectString(t, "Username:")
		exp.Send(t, "admin\n")
		exp.ExpectString(t, "Password")
		exp.Send(t, "password\n")
		exp.ExpectString(t, "Confirm")
		exp.Send(t, "password\n")
		exp.ExpectString(t, "Logged in as")

		assert.NilError(t, g.Wait())
	})

	runStep(t, "login updated infra config", func(t *testing.T) {
		cfg, err := readConfig()
		assert.NilError(t, err)

		expected := []ClientHostConfig{
			{
				Name:          "admin",
				AccessKey:     "any-access-key",
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

	term := vt10x.New(vt10x.WithWriter(tty))
	console, err := expect.NewConsole(
		expect.WithDefaultTimeout(2*time.Second),
		expect.WithStdout(os.Stdout),
		expect.WithStdin(pseudoTY),
		expect.WithStdout(term),
		expect.WithCloser(pseudoTY, tty))
	assert.NilError(t, err)
	t.Cleanup(func() {
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
