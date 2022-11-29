package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/uid"
)

func TestMain(m *testing.M) {
	// Many tests in this package will modify the home directory. Every test
	// should call t.Setenv, but as a safeguard setup some env vars.
	_ = os.Setenv("HOME", "/this test forgot to t.SetEnv(HOME, ...)")
	_ = os.Setenv("USERPROFILE", "/this test forgot to t.SetEnv(USERPROFILE, ...)")
	_ = os.Setenv("KUBECONFIG", "/this test forgot to t.SetEnv(KUBECONFIG, ...)")

	// Default to not running the agent in tests, because background processes
	// are difficult to manage.
	_ = os.Setenv("INFRA_NO_AGENT", "true")

	os.Exit(m.Run())
}

// destinationCA is a well formed certificate that can be used to create
// a destination in tests.
var destinationCA = api.PEM(`-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUETRDuZAQHGhiH11GNsXn16n9t48wDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjA0MTIyMTAzMDhaFw0yNDA0
MTEyMTAzMDhaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQC6GBhadDfSLlgXsL7sWOExlOboQYAQh2pjfjUjMjgW
ZNQhguRnA4iDCXeBVJnrlTxJvBUJpZ5Wd3h6Tp3Yf9o8teJCvRqX99uuD1P/4P2O
gcEpiXxEmnAgsNeZfUVCQJhhHM9BGUEn+3FRL6yuSVi+6F6Xu+FmQ0xERu3M7Gv8
dtXdn1y8rSxNPME8+VFAon47phGAa4aACZOo5dqbfkKNSJlLK2B7B6MYuVtI14kk
GuVtLy/sEJlH1ZROPE7zeyh7ZXsGXr8O/sCmXTZNAe98mTUxZX0IxT6drgcwzFdK
6BJNAxvgBsJltpAGrVo+m+pm8HWmnAS0NTXYPUofYD0NAgMBAAGjUzBRMB0GA1Ud
DgQWBBT/khk5FFePHZ7v5tT/3QeHggVHETAfBgNVHSMEGDAWgBT/khk5FFePHZ7v
5tT/3QeHggVHETAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCx
XQyY89xU9XA29JSn96oOQQRNVDl1PmhNiIrJI7FCn5vK1+do00i5teO8mAb49IMt
DGA8pCAllFTiz6ibf8IuVnCype4lLbJ19am648IllV97Dwo/gnlF08ozWai2mx6l
5rOqg0YSpEWB88xbulVPWpjwAzYsXh8Y7kem7TXd9MICsIkl1+BXjgG7LSaIwa60
swYRJSf2bpBsW0Hiqx6WlLUETieVJF9gld0FZSG5Vix0y0IdPEZD5ACbM5G2X4QB
XlW7KilKI5YkcszGoPB4RePiHsH+7trf7l8IQq5r5kRq7SKsZ41BI6s1E1PQVW93
7Crix1N6DuA9FeukBz2M
-----END CERTIFICATE-----`)

// newTestClientConfig returns a ClientConfig that can be used to test CLI
// commands. Most CLI commands require a login first, which saves a ClientConfig
// to a file.
// newTestClientConfig provides a reasonable default for most cases, removing
// the need to perform a full login. The returned value may be modified, and then
// should be saved to a file with writeConfig.
// If any fields in identity are not set, they will be set to default values.
func newTestClientConfig(srv *httptest.Server, user api.User) ClientConfig {
	if user.Name == "" {
		user.Name = "testuser@example.com"
	}
	if user.ID == 0 {
		user.ID = uid.New()
	}
	return ClientConfig{
		ClientConfigVersion: clientConfigVersion,
		Hosts: []ClientHostConfig{
			{
				UserID:             user.ID,
				Name:               user.Name,
				Host:               srv.Listener.Addr().String(),
				TrustedCertificate: string(certs.PEMEncodeCertificate(srv.Certificate().Raw)),
				AccessKey:          "the-access-key",
				Expires:            api.Time(time.Now().Add(time.Hour)),
				Current:            true,
			},
		},
	}
}

func newExpiredTestClientConfig(srv *httptest.Server, user api.User) ClientConfig {
	if user.Name == "" {
		user.Name = "testuser@example.com"
	}
	if user.ID == 0 {
		user.ID = uid.New()
	}
	return ClientConfig{
		ClientConfigVersion: clientConfigVersion,
		Hosts: []ClientHostConfig{
			{
				UserID:             user.ID,
				Name:               user.Name,
				Host:               srv.Listener.Addr().String(),
				TrustedCertificate: string(certs.PEMEncodeCertificate(srv.Certificate().Raw)),
				AccessKey:          "the-access-key",
				Expires:            api.Time(time.Now().Add(-1 * time.Hour)),
				Current:            true,
			},
		},
	}
}

func newClearedTestClientConfig(srv *httptest.Server) ClientConfig {
	return ClientConfig{
		ClientConfigVersion: clientConfigVersion,
		Hosts: []ClientHostConfig{
			{
				UserID:             0,
				Name:               "",
				Host:               srv.Listener.Addr().String(),
				TrustedCertificate: string(certs.PEMEncodeCertificate(srv.Certificate().Raw)),
				AccessKey:          "",
				Expires:            api.Time(time.Now().Add(time.Hour)),
				Current:            true,
			},
		},
	}
}

func TestInvalidSessions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows

	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	t.Setenv("KUBECONFIG", filepath.Join(home, "config"))

	srv := httptest.NewTLSServer(http.HandlerFunc(nil))
	t.Cleanup(srv.Close)

	t.Run("Expired session", func(t *testing.T) {
		cfg := newExpiredTestClientConfig(srv, api.User{ID: uid.New()})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		err = clearKubeconfig()
		assert.NilError(t, err)

		err = Run(context.Background(), "destinations", "list")
		assert.ErrorContains(t, err, "Access key is expired, please `infra login` again")
	})

	t.Run("Logged out session", func(t *testing.T) {
		cfg := newClearedTestClientConfig(srv)
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		err = clearKubeconfig()
		assert.NilError(t, err)

		err = Run(context.Background(), "destinations", "list")
		assert.ErrorContains(t, err, "Missing access key, must `infra login` or set INFRA_ACCESS_KEY in your environment")
	})
}

func TestRootCmd_UsageTemplate(t *testing.T) {
	ctx := context.Background()
	cmd := NewRootCmd(newCLI(ctx))

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	assert.NilError(t, cmd.Usage())

	golden.Assert(t, buf.String(), "expected-usage")
}

// TestCmdDoesNotUsePersistentPreRun because if any subcommand sets a
// PersistentPreRun it will override the one set by the root command.
func TestCmdDoesNotUsePersistentPreRun(t *testing.T) {
	ctx := context.Background()
	cli := newCLI(ctx)
	cmd := NewRootCmd(cli)

	walkCommands(cmd, func(child *cobra.Command) {
		if child.PersistentPreRun != nil || child.PersistentPreRunE != nil {
			t.Errorf("command %q should not use PersistentPreRun", child.CommandPath())
		}
	})
}

// walkCommands walks the Command in depth first order, and calls fn for
// every child command in the tree.
func walkCommands(cmd *cobra.Command, fn func(child *cobra.Command)) {
	for _, child := range cmd.Commands() {
		fn(child)
		if len(child.Commands()) > 0 {
			walkCommands(child, fn)
		}
	}
}
