package cmd

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/uid"
)

var anyUID = uid.ID(99)

// runAndWait runs fn in a goroutine and adds a t.Cleanup function to wait for
// the goroutine to exit before ending cleanup. runAndWait is used to ensure
// that the goroutine exits before a new test starts.
func runAndWait(ctx context.Context, t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		t.Helper()
		assert.Check(t, fn(ctx))
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func TestLoginCmd_Options(t *testing.T) {
	dir := setupEnv(t)

	opts := defaultServerOptions(dir)
	setupServerOptions(t, &opts)
	opts.Config.Users = []server.User{
		{
			Name:     "admin@example.com",
			Password: "p4ssw0rd",
		},
	}
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx := context.Background()
	runAndWait(ctx, t, srv.Run)

	runStep(t, "login without background agent", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)

		t.Setenv("INFRA_PASSWORD", "p4ssw0rd")

		// TODO: remove --skip-tls-verify
		err := Run(ctx, "login", srv.Addrs.HTTPS.String(), "--skip-tls-verify", "--no-agent", "--user", "admin@example.com")
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
				AccessKey:     "any access key",
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
		cmpIDNotZero),
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

var cmpIDNotZero = cmp.Comparer(func(x, y uid.ID) bool {
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

func setupServerOptions(t *testing.T, opts *server.Options) {
	t.Helper()

	opts.Addr = server.ListenerOptions{HTTPS: "127.0.0.1:0", HTTP: "127.0.0.1:0"}

	key, err := os.ReadFile("testdata/pki/localhost.key")
	assert.NilError(t, err)
	opts.TLS.PrivateKey = string(key)

	cert, err := os.ReadFile("testdata/pki/localhost.crt")
	assert.NilError(t, err)
	opts.TLS.Certificate = types.StringOrFile(cert)

	pgDriver := database.PostgresDriver(t, "_cmd")
	opts.DBConnectionString = pgDriver.DSN
}

func TestLoginCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // for windows

	t.Run("without required arguments", func(t *testing.T) {
		err := Run(context.Background(), "login")
		assert.ErrorContains(t, err, "INFRA_SERVER to be set")
	})
}

func TestLoginCmd_UserPass(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // for windows
	kubeConfigPath := filepath.Join(home, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	t.Run("without username flag and without tty", func(t *testing.T) {
		err := Run(context.Background(), "login", "example.infrahq.com")
		assert.ErrorContains(t, err, "INFRA_USER to be set")
	})

	t.Run("without password flag and without tty", func(t *testing.T) {
		err := Run(context.Background(), "login", "example.infrahq.com", "--user", "foo")
		assert.ErrorContains(t, err, "INFRA_PASSWORD to be set")
	})

	t.Run("logs in", func(t *testing.T) {
		handler := func(resp http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/api/login" {

				var loginRequest api.LoginRequest
				err := json.NewDecoder(req.Body).Decode(&loginRequest)
				assert.Check(t, err)
				assert.Equal(t, loginRequest.PasswordCredentials.Name, "admin@example.com")
				assert.Equal(t, loginRequest.PasswordCredentials.Password, "p4ssw0rd")

				res := &api.LoginResponse{
					UserID:                 uid.New(),
					Name:                   "foo@gmail.com",
					AccessKey:              "abc.xyz",
					OrganizationName:       "Default",
					PasswordUpdateRequired: false,
					Expires:                api.Time(time.Now().UTC().Add(time.Hour * 24)),
				}
				err = json.NewEncoder(resp).Encode(res)
				assert.Check(t, err)
			}
		}

		t.Setenv("INFRA_USER", "admin@example.com")
		t.Setenv("INFRA_PASSWORD", "p4ssw0rd")

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		ctx, bufs := PatchCLI(context.Background())

		err := Run(ctx, "login", srv.Listener.Addr().String(), "--tls-trusted-fingerprint", certs.Fingerprint(srv.Certificate().Raw))
		assert.NilError(t, err)

		assert.Assert(t, strings.Contains(bufs.Stderr.String(), "Logged in as"))
		assert.Assert(t, strings.Contains(bufs.Stderr.String(), "foo@gmail.com"))
	})
}

func TestLoginCmd_TLSVerify(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(dir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	opts := defaultServerOptions(dir)
	setupServerOptions(t, &opts)
	opts.Users = []server.User{
		{Name: "admin@example.com", Password: "p4ssw0rd"},
	}
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx := context.Background()
	runAndWait(ctx, t, srv.Run)

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

		t.Setenv("INFRA_PASSWORD", "p4ssw0rd")

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			// TODO: why isn't this working without --non-interactive=false? the other test works
			return Run(ctx, "login", "--non-interactive=false", "--user", "admin@example.com", srv.Addrs.HTTPS.String())
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

		t.Setenv("INFRA_PASSWORD", "p4ssw0rd")

		err = Run(ctx, "login", "--user", "admin@example.com", srv.Addrs.HTTPS.String())
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

		t.Setenv("INFRA_PASSWORD", "p4ssw0rd")

		err = Run(ctx, "login",
			"--tls-trusted-cert", "testdata/pki/localhost.crt",
			"--user", "admin@example.com",
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
		t.Setenv("INFRA_SERVER", srv.Addrs.HTTPS.String())

		err := Run(ctx, "logout", "--clear")
		assert.NilError(t, err)

		cert, err := os.ReadFile("testdata/pki/localhost.crt")
		assert.NilError(t, err)

		block, _ := pem.Decode(cert)
		fingerprint := certs.Fingerprint(block.Bytes)

		t.Setenv("INFRA_USER", "admin@example.com")
		t.Setenv("INFRA_PASSWORD", "p4ssw0rd")

		err = Run(ctx, "login", "--tls-trusted-fingerprint", fingerprint)
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

		t.Setenv("INFRA_PASSWORD", "p4ssw0rd")

		err = Run(ctx, "login",
			"--tls-trusted-fingerprint", "BA::D0::FF",
			"--user", "admin@example.com",
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
