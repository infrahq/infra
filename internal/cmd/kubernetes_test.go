package cmd

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/infrahq/infra/api"
)

func TestWriteKubeconfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KUBECONFIG", filepath.Join(home, "nonexistent", "kubeconfig"))

	user := api.User{Name: "user"}
	destinations := []api.Destination{
		{
			Name: "cluster",
			Connection: api.DestinationConnection{
				URL: "cluster.example.com",
				CA:  destinationCA,
			},
		},
	}
	grants := []api.Grant{
		{
			Resource: "cluster",
		},
	}

	err := writeKubeconfig(&user, destinations, grants)
	assert.NilError(t, err)

	expected := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"infra:cluster": {
				Server:                   "https://cluster.example.com",
				CertificateAuthorityData: destinationCA,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"infra:cluster": {
				AuthInfo: "user",
				Cluster:  "infra:cluster",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"user": {},
		},
	}

	actual, err := clientConfig().RawConfig()
	assert.NilError(t, err)

	assert.DeepEqual(t, expected, actual,
		cmpopts.EquateEmpty(),
		cmpopts.IgnoreFields(clientcmdapi.Cluster{}, "LocationOfOrigin"),
		cmpopts.IgnoreFields(clientcmdapi.Context{}, "LocationOfOrigin"),
		cmpopts.IgnoreFields(clientcmdapi.AuthInfo{}, "LocationOfOrigin"),
		cmpopts.IgnoreFields(clientcmdapi.AuthInfo{}, "Exec"),
	)
}
