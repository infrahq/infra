package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/connector"
	"github.com/infrahq/infra/uid"
)

func TestMain(m *testing.M) {
	// Many tests in this package will modify the home directory. Every test
	// should call t.Setenv, but as a safeguard setup some env vars.
	_ = os.Setenv("HOME", "/this test forgot to t.SetEnv(HOME, ...)")
	_ = os.Setenv("USERPROFILE", "/this test forgot to t.SetEnv(USERPROFILE, ...)")
	_ = os.Setenv("KUBECONFIG", "/this test forgot to t.SetEnv(KUBECONFIG, ...)")
	os.Exit(m.Run())
}

func TestCanonicalPath(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	t.Setenv("USERPROFILE", "/home/user")
	wd, err := filepath.EvalSymlinks(t.TempDir())
	assert.NilError(t, err)

	env.ChangeWorkingDir(t, wd)

	type testCase struct {
		path     string
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := canonicalPath(tc.path)
		assert.NilError(t, err)
		assert.Equal(t, tc.expected, actual)
	}

	testCases := []testCase{
		{path: "/already/abs", expected: "/already/abs"},
		{path: "relative/no/dot", expected: wd + "/relative/no/dot"},
		{path: "./relative/dot", expected: wd + "/relative/dot"},
		{path: "$HOME/dir", expected: "/home/user/dir"},
		{path: "${HOME}/dir", expected: "/home/user/dir"},
		{path: "/not/$HOMEFOO/dir", expected: "/not/dir"},
		{path: "$HOMEFOO/dir", expected: "/dir"},
		{path: "~/config", expected: "/home/user/config"},
		{path: "~user/config", expected: wd + "/~user/config"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("in=%v out=%v", tc.path, tc.expected), func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestUse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows

	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	t.Setenv("KUBECONFIG", filepath.Join(home, "config"))

	userID := uid.New()
	destinationID := uid.New()

	setup := func(t *testing.T) *ClientConfig {
		handler := func(resp http.ResponseWriter, req *http.Request) {
			query := req.URL.Query()
			switch {
			case req.URL.Path == "/api/destinations":
				destinations := api.ListResponse[api.Destination]{
					Items: []api.Destination{
						{
							ID:       destinationID,
							UniqueID: "uniqueID",
							Name:     "cluster",
							Connection: api.DestinationConnection{
								URL: "kubernetes.docker.local",
								CA:  destinationCA,
							},
						},
					},
				}

				bytes, err := json.Marshal(destinations)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			case req.URL.Path == "/api/grants":
				grants := api.ListResponse[api.Grant]{
					Items: []api.Grant{
						{
							ID:        uid.New(),
							User:      userID,
							Resource:  "cluster",
							Privilege: "admin",
						},
						{
							ID:        uid.New(),
							User:      userID,
							Resource:  "cluster.namespace",
							Privilege: "admin",
						},
					},
				}

				bytes, err := json.Marshal(grants)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			case req.URL.Path == "/api/groups" && query.Get("userID") == userID.String():
				groups := api.ListResponse[api.Group]{}
				bytes, err := json.Marshal(groups)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			case req.URL.Path == fmt.Sprintf("/api/users/%s", userID):
				user := api.User{
					ID:   userID,
					Name: "testuser@example.com",
				}

				bytes, err := json.Marshal(user)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			default:
				resp.WriteHeader(http.StatusBadRequest)
			}
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := newTestClientConfig(srv, api.User{ID: userID})
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		err = clearKubeconfig()
		assert.NilError(t, err)

		return &cfg
	}

	t.Run("UseCluster", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "cluster")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 2)
		assert.Equal(t, len(kubeconfig.Contexts), 2)
		assert.Equal(t, len(kubeconfig.AuthInfos), 1)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster")
		assert.Assert(t, is.Contains(kubeconfig.AuthInfos, "testuser@example.com"))
	})

	t.Run("UseNamespace", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "cluster.namespace")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 2)
		assert.Equal(t, len(kubeconfig.Contexts), 2)
		assert.Equal(t, len(kubeconfig.AuthInfos), 1)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster:namespace")
		assert.Assert(t, is.Contains(kubeconfig.AuthInfos, "testuser@example.com"))
	})

	t.Run("InfraUse", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "infra:cluster")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster")

		err = Run(context.Background(), "use", "infra:cluster.namespace")
		assert.NilError(t, err)

		kubeconfig, err = clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster:namespace")
	})

	t.Run("UseUnknown", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "unknown")
		assert.ErrorContains(t, err, "context not found")
	})

	t.Run("missing argument", func(t *testing.T) {
		err := Run(context.Background(), "use")
		assert.ErrorContains(t, err, `"infra use" requires exactly 1 argument`)
		assert.ErrorContains(t, err, `Usage:  infra use`)
	})
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
		Version: "0.3",
		Hosts: []ClientHostConfig{
			{
				PolymorphicID: uid.NewIdentityPolymorphicID(user.ID),
				Name:          user.Name,
				Host:          srv.Listener.Addr().String(),
				SkipTLSVerify: true,
				AccessKey:     "the-access-key",
				Expires:       api.Time(time.Now().Add(time.Hour)),
				Current:       true,
			},
		},
	}
}

func TestConnectorCmd(t *testing.T) {
	var actual connector.Options
	patchRunConnector(t, func(ctx context.Context, options connector.Options) error {
		actual = options
		return nil
	})

	content := `
server: the-server
name: the-name
accessKey: /var/run/secrets/key
caCert: /path/to/cert
caKey: /path/to/key
skipTLSVerify: true
`

	dir := fs.NewDir(t, t.Name(), fs.WithFile("config.yaml", content))

	ctx := context.Background()
	err := Run(ctx, "connector", "-f", dir.Join("config.yaml"))
	assert.NilError(t, err)

	expected := connector.Options{
		Name:          "the-name",
		Server:        "the-server",
		AccessKey:     "/var/run/secrets/key",
		CACert:        "/path/to/cert",
		CAKey:         "/path/to/key",
		SkipTLSVerify: true,
	}
	assert.DeepEqual(t, actual, expected)
}

func patchRunConnector(t *testing.T, fn func(context.Context, connector.Options) error) {
	orig := runConnector
	runConnector = fn
	t.Cleanup(func() {
		runConnector = orig
	})
}

func TestConnectorCmd_NoFlagDefaults(t *testing.T) {
	cmd := newConnectorCmd()
	flags := cmd.Flags()
	err := flags.Parse(nil)
	assert.NilError(t, err)

	msg := "The default value of flags on the 'infra connector' command will be ignored. " +
		"Set a default value in defaultConnectorOptions instead."
	flags.VisitAll(func(flag *pflag.Flag) {
		if sv, ok := flag.Value.(pflag.SliceValue); ok {
			if len(sv.GetSlice()) > 0 {
				t.Fatalf("Flag --%v uses non-zero value %v. %v", flag.Name, flag.Value, msg)
			}
			return
		}

		v := reflect.Indirect(reflect.ValueOf(flag.Value))
		if !v.IsZero() {
			t.Fatalf("Flag --%v uses non-zero value %v. %v", flag.Name, flag.Value, msg)
		}
	})
}
