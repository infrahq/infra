package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestLogout(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir) // for windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(homeDir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	setup := func(t *testing.T) (ClientConfig, *int32) {
		var count int32
		handler := func(resp http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/v1/logout" {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			atomic.AddInt32(&count, 1)
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write([]byte(`{}`)) // API client requires a JSON response
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)
		srv2 := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv2.Close)

		cfg := ClientConfig{
			Version: "0.3",
			Hosts: []ClientHostConfig{
				{
					Name:          "host1",
					Host:          srv.Listener.Addr().String(),
					AccessKey:     "the-access-key",
					SkipTLSVerify: true,
				},
				{
					Name:          "host2",
					Host:          srv2.Listener.Addr().String(),
					AccessKey:     "the-access-key",
					SkipTLSVerify: true,
				},
			},
		}
		err := writeConfig(&cfg)
		assert.NilError(t, err)

		kubeCfg := clientcmdapi.Config{
			Clusters: map[string]*clientcmdapi.Cluster{
				"keep:not-infra": {Server: "https://keep:8080"},
				"infra:prod":     {Server: "https://infraprod:8080"},
			},
			Contexts: map[string]*clientcmdapi.Context{
				"keep:not-infra": {Cluster: "keep:not-infra"},
				"infra:prod":     {Cluster: "infra:prod"},
			},
			AuthInfos: map[string]*clientcmdapi.AuthInfo{
				"keep:not-infra": {Token: "keep-token"},
				"infra:prod":     {Token: "infra-token"},
			},
		}
		err = clientcmd.WriteToFile(kubeCfg, kubeConfigPath)
		assert.NilError(t, err)
		return cfg, &count
	}

	expectedKubeCfg := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"keep:not-infra": {Server: "https://keep:8080", LocationOfOrigin: kubeConfigPath},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"keep:not-infra": {Cluster: "keep:not-infra", LocationOfOrigin: kubeConfigPath},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"keep:not-infra": {Token: "keep-token", LocationOfOrigin: kubeConfigPath},
		},
	}

	t.Run("default", func(t *testing.T) {
		cfg, count := setup(t)
		err := newLogoutCmd().Execute()
		assert.NilError(t, err)

		assert.Equal(t, int32(2), atomic.LoadInt32(count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)

		expected := cfg
		expected.Hosts[0].AccessKey = ""
		expected.Hosts[1].AccessKey = ""
		assert.DeepEqual(t, &expected, updatedCfg)

		updatedKubeCfg, err := clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.DeepEqual(t, expectedKubeCfg, updatedKubeCfg, cmpopts.EquateEmpty())
	})

	t.Run("with purge", func(t *testing.T) {
		_, count := setup(t)
		cmd := newLogoutCmd()
		cmd.SetArgs([]string{"--purge"})
		err := cmd.Execute()
		assert.NilError(t, err)

		assert.Equal(t, int32(2), atomic.LoadInt32(count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)

		expected := ClientConfig{Version: "0.3"}
		assert.DeepEqual(t, &expected, updatedCfg)

		updatedKubeCfg, err := clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.DeepEqual(t, expectedKubeCfg, updatedKubeCfg, cmpopts.EquateEmpty())
	})
}
