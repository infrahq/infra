package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

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

	setup := func(t *testing.T) *ClientConfig {
		handler := func(resp http.ResponseWriter, req *http.Request) {
			query := req.URL.Query()
			switch {
			case req.URL.Path == "/api/destinations":
				destinations := api.ListResponse[api.Destination]{
					Items: []api.Destination{
						{
							ID:        destinationID,
							UniqueID:  "uniqueID",
							Name:      "cluster",
							Connected: true,
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
							ID:              uid.New(),
							User:            userID,
							DestinationName: "cluster",
							Privilege:       "admin",
						},
						{
							ID:                  uid.New(),
							User:                userID,
							DestinationName:     "cluster",
							DestinationResource: "namespace",
							Privilege:           "admin",
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

		assert.Equal(t, len(kubeconfig.Clusters), 1)
		assert.Equal(t, len(kubeconfig.Contexts), 1)
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

		assert.Equal(t, len(kubeconfig.Clusters), 1)
		assert.Equal(t, len(kubeconfig.Contexts), 1)
		assert.Equal(t, len(kubeconfig.AuthInfos), 1)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster")
		assert.Equal(t, kubeconfig.Contexts[kubeconfig.CurrentContext].Namespace, "namespace")
		assert.Assert(t, is.Contains(kubeconfig.AuthInfos, "testuser@example.com"))
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

	t.Run("use cluster does not change namespace", func(t *testing.T) {
		setup(t)

		err := Run(context.Background(), "use", "cluster.namespace")
		assert.NilError(t, err)

		kubeconfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 1)
		assert.Equal(t, len(kubeconfig.Contexts), 1)
		assert.Equal(t, len(kubeconfig.AuthInfos), 1)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster")
		assert.Equal(t, kubeconfig.Contexts[kubeconfig.CurrentContext].Namespace, "namespace")
		assert.Assert(t, is.Contains(kubeconfig.AuthInfos, "testuser@example.com"))

		err = Run(context.Background(), "use", "cluster")
		assert.NilError(t, err)

		kubeconfig, err = clientConfig().RawConfig()
		assert.NilError(t, err)

		assert.Equal(t, len(kubeconfig.Clusters), 1)
		assert.Equal(t, len(kubeconfig.Contexts), 1)
		assert.Equal(t, len(kubeconfig.AuthInfos), 1)
		assert.Equal(t, kubeconfig.CurrentContext, "infra:cluster")
		assert.Equal(t, kubeconfig.Contexts[kubeconfig.CurrentContext].Namespace, "namespace")
		assert.Assert(t, is.Contains(kubeconfig.AuthInfos, "testuser@example.com"))
	})
}
