package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/infrahq/infra/api"
)

func TestWriteKubeconfig(t *testing.T) {
	user := api.User{Name: "user"}
	destinations := []api.Destination{
		{
			Name:      "connected",
			Connected: true,
			Connection: api.DestinationConnection{
				URL: "connected.example.com",
				CA:  destinationCA,
			},
		},
		{
			Name:      "pending",
			Connected: true,
			Connection: api.DestinationConnection{
				CA: destinationCA,
			},
		},
		{
			Name:      "disconnected",
			Connected: false,
			Connection: api.DestinationConnection{
				URL: "disconnected.example.com",
				CA:  destinationCA,
			},
		},
	}

	run := func(t *testing.T, grants ...api.Grant) clientcmdapi.Config {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		kubeConfigPath := filepath.Join(home, "nonexistent", "kubeconfig")
		t.Setenv("KUBECONFIG", kubeConfigPath)

		err := writeKubeconfig(&user, destinations, grants)
		assert.NilError(t, err)

		configFileStat, err := os.Stat(kubeConfigPath)
		assert.NilError(t, err)
		assert.Equal(t, int(configFileStat.Mode().Perm()), 0o600)

		kubeConfig, err := clientConfig().RawConfig()
		assert.NilError(t, err)

		return kubeConfig
	}

	expectedClusters := map[string]*clientcmdapi.Cluster{
		"infra:connected": {
			Server:                   "https://connected.example.com",
			CertificateAuthorityData: []byte(destinationCA),
		},
	}

	expectedAuthInfos := map[string]*clientcmdapi.AuthInfo{
		"user": {},
	}

	kubeConfigCmpOpts := []cmp.Option{
		cmpopts.EquateEmpty(),
		cmpopts.IgnoreFields(clientcmdapi.Context{}, "LocationOfOrigin"),
		cmpopts.IgnoreFields(clientcmdapi.Cluster{}, "LocationOfOrigin"),
		cmpopts.IgnoreFields(clientcmdapi.AuthInfo{}, "LocationOfOrigin"),
		cmpopts.IgnoreFields(clientcmdapi.AuthInfo{}, "Exec"),
	}

	t.Run("OneNamespace", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo:  "user",
				Cluster:   "infra:connected",
				Namespace: "namespace",
			},
		}

		actual := run(t, api.Grant{Resource: "connected.namespace"})

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})

	t.Run("MultipleNamespaces", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo:  "user",
				Cluster:   "infra:connected",
				Namespace: "namespace",
			},
		}

		grants := []api.Grant{
			{Resource: "connected.namespace"},
			{Resource: "connected.namespace2"},
			{Resource: "connected.namespace3"},
		}

		actual := run(t, grants...)

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})

	t.Run("DefaultNamespace", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo: "user",
				Cluster:  "infra:connected",
			},
		}

		grants := []api.Grant{
			{Resource: "connected.default"},
		}

		actual := run(t, grants...)

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})

	t.Run("DefaultAndMultipleNamespaces", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo: "user",
				Cluster:  "infra:connected",
			},
		}

		grants := []api.Grant{
			{Resource: "connected.namespace"},
			{Resource: "connected.namespace2"},
			{Resource: "connected.default"},
			{Resource: "connected.namespace3"},
		}

		actual := run(t, grants...)

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})

	t.Run("Cluster", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo: "user",
				Cluster:  "infra:connected",
			},
		}

		actual := run(t, api.Grant{Resource: "connected"})

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})

	t.Run("ClusterAndDefaultNamespace", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo: "user",
				Cluster:  "infra:connected",
			},
		}

		grants := []api.Grant{
			{Resource: "connected.default"},
			{Resource: "connected"},
		}

		actual := run(t, grants...)

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})

	t.Run("ClusterAndMultipleNamespaces", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo: "user",
				Cluster:  "infra:connected",
			},
		}

		grants := []api.Grant{
			{Resource: "connected.namespace"},
			{Resource: "connected.namespace2"},
			{Resource: "connected.namespace3"},
			{Resource: "connected"},
		}

		actual := run(t, grants...)

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})

	t.Run("OmitUnavailableClusters", func(t *testing.T) {
		expectedContexts := map[string]*clientcmdapi.Context{
			"infra:connected": {
				AuthInfo: "user",
				Cluster:  "infra:connected",
			},
		}

		grants := []api.Grant{
			{Resource: "connected"},
			{Resource: "disconnected"},
			{Resource: "pending"},
		}

		actual := run(t, grants...)

		assert.DeepEqual(t, actual.Contexts, expectedContexts, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.Clusters, expectedClusters, kubeConfigCmpOpts...)
		assert.DeepEqual(t, actual.AuthInfos, expectedAuthInfos, kubeConfigCmpOpts...)
	})
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
			Name:      "cluster",
			Connected: true,
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

func TestSafelyWriteConfigToFile(t *testing.T) {
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
				Namespace: "default",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"user": {},
		},
	}

	err := safelyWriteConfigToFile(expected, kubeconfig)
	assert.NilError(t, err)

	// check that the file is written to
	actual, err := clientConfig().RawConfig()
	assert.NilError(t, err)
	assert.Equal(t, actual.Contexts["infra:cluster:default"].Namespace, "default")

	// check that the temp file is gone
	files, err := ioutil.ReadDir(home)
	assert.NilError(t, err)

	for _, file := range files {
		// if the file name contains this string it was the temp file
		assert.Assert(t, !strings.Contains(file.Name(), "infra-kube-config"))
	}
}
