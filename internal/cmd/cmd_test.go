package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func TestUse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows

	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	t.Setenv("KUBECONFIG", filepath.Join(home, "config"))

	userID := uid.New()
	destinationID := uid.New()
	ca := `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`

	setup := func(t *testing.T) *ClientConfig {
		handler := func(resp http.ResponseWriter, req *http.Request) {
			switch {
			case req.URL.Path == "/v1/destinations":
				destinations := []api.Destination{
					{
						ID:       destinationID,
						UniqueID: "uniqueID",
						Name:     "kubernetes.cluster",
						Connection: api.DestinationConnection{
							URL: "kubernetes.docker.local",
							CA:  ca,
						},
					},
				}

				bytes, err := json.Marshal(destinations)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			case req.URL.Path == fmt.Sprintf("/v1/identities/%s/grants", userID):
				grants := []api.Grant{
					{
						ID:        uid.New(),
						Subject:   uid.NewIdentityPolymorphicID(userID),
						Resource:  "kubernetes.cluster",
						Privilege: "admin",
					},
					{
						ID:        uid.New(),
						Subject:   uid.NewIdentityPolymorphicID(userID),
						Resource:  "kubernetes.cluster.namespace",
						Privilege: "admin",
					},
				}

				bytes, err := json.Marshal(grants)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			case req.URL.Path == fmt.Sprintf("/v1/identities/%s/groups", userID):
				groups := []api.Group{}

				bytes, err := json.Marshal(groups)
				assert.NilError(t, err)

				_, err = resp.Write(bytes)
				assert.NilError(t, err)
			default:
				resp.WriteHeader(http.StatusBadRequest)
			}
		}

		svc := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(svc.Close)

		cfg := ClientConfig{
			Version: "0.3",
			Hosts: []ClientHostConfig{
				{
					PolymorphicID: uid.NewIdentityPolymorphicID(userID),
					Name:          "test",
					Host:          svc.Listener.Addr().String(),
					SkipTLSVerify: true,
					AccessKey:     "access-key",
					Expires:       api.Time(time.Now().Add(time.Hour)),
					Current:       true,
				},
			},
		}

		err := writeConfig(&cfg)
		assert.NilError(t, err)

		err = clearKubeconfig()
		assert.NilError(t, err)

		return &cfg
	}

	t.Run("UseClusterWithoutPrefix", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "cluster")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 2)
		assert.Equal(t, len(kubeconfig.Contexts), 2)
		assert.Equal(t, len(kubeconfig.AuthInfos), 2)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster")
	})

	t.Run("UseClusterWithPrefix", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "kubernetes.cluster")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 2)
		assert.Equal(t, len(kubeconfig.Contexts), 2)
		assert.Equal(t, len(kubeconfig.AuthInfos), 2)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster")
	})

	t.Run("UseNamespaceWithoutPrefix", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "cluster.namespace")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 2)
		assert.Equal(t, len(kubeconfig.Contexts), 2)
		assert.Equal(t, len(kubeconfig.AuthInfos), 2)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster:namespace")
	})

	t.Run("UseNamespaceWithPrefix", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "kubernetes.cluster.namespace")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 2)
		assert.Equal(t, len(kubeconfig.Contexts), 2)
		assert.Equal(t, len(kubeconfig.AuthInfos), 2)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster:namespace")
	})

	t.Run("UseUnknown", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "unknown")
		assert.ErrorContains(t, err, "context not found")
	})
}
