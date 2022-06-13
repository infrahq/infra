package cmd

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"k8s.io/client-go/tools/clientcmd"
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
				CertificateAuthorityData: []byte(destinationCA),
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

func TestWriteKubeconfig_UserNamespaceOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	kubeconfig := filepath.Join(home, "kubeconfig")
	t.Setenv("KUBECONFIG", kubeconfig)

	expected := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"infra:cluster": {
				Server:                   "https://cluster.example.com",
				CertificateAuthorityData: []byte(destinationCA),
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"infra:cluster": {
				AuthInfo:  "user",
				Cluster:   "infra:cluster",
				Namespace: "override",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"user": {},
		},
	}

	err := clientcmd.WriteToFile(expected, kubeconfig)
	assert.NilError(t, err)

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

	err = writeKubeconfig(&user, destinations, grants)
	assert.NilError(t, err)

	actual, err := clientConfig().RawConfig()
	assert.NilError(t, err)
	assert.Equal(t, actual.Contexts["infra:cluster"].Namespace, "override")
}

func TestWriteKubeconfig_UserNamespaceOverrideResetNamespacedContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	kubeconfig := filepath.Join(home, "kubeconfig")
	t.Setenv("KUBECONFIG", kubeconfig)

	expected := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"infra:cluster:default": {
				Server:                   "https://cluster.example.com",
				CertificateAuthorityData: []byte(destinationCA),
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"infra:cluster:default": {
				AuthInfo:  "user",
				Cluster:   "infra:cluster",
				Namespace: "override",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"user": {},
		},
	}

	err := clientcmd.WriteToFile(expected, kubeconfig)
	assert.NilError(t, err)

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
			Resource: "cluster.default",
		},
	}

	err = writeKubeconfig(&user, destinations, grants)
	assert.NilError(t, err)

	actual, err := clientConfig().RawConfig()
	assert.NilError(t, err)
	assert.Equal(t, actual.Contexts["infra:cluster:default"].Namespace, "default")
}
