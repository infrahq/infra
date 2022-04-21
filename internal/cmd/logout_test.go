package cmd

import (
	"context"
	"fmt"
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

type testFields struct {
	config     ClientConfig
	count      *int32
	serverURLs []string
}

func TestLogout(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir) // for windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(homeDir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	setup := func(t *testing.T) testFields {
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
					Name:          "user1",
					Host:          srv.Listener.Addr().String(),
					AccessKey:     "the-access-key",
					PolymorphicID: "pid1",
					SkipTLSVerify: true,
					Current:       true,
				},
				{
					Name:          "user2",
					Host:          srv2.Listener.Addr().String(),
					AccessKey:     "the-access-key",
					PolymorphicID: "pid2",
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
		return testFields{
			config:     cfg,
			count:      &count,
			serverURLs: []string{srv.Listener.Addr().String(), srv2.Listener.Addr().String()},
		}
	}

	setupError := func(t *testing.T) testFields {
		var count int32
		handler := func(resp http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/v1/logout" {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			atomic.AddInt32(&count, 1)
			resp.WriteHeader(http.StatusInternalServerError)
			_, _ = resp.Write([]byte(`{}`)) // API client requires a JSON response
		}

		srv := httptest.NewTLSServer(http.HandlerFunc(handler))
		t.Cleanup(srv.Close)

		cfg := ClientConfig{
			Version: "0.3",
			Hosts: []ClientHostConfig{
				{
					Name:          "user1",
					Host:          srv.Listener.Addr().String(),
					AccessKey:     "the-access-key",
					PolymorphicID: "pid1",
					SkipTLSVerify: true,
					Current:       true,
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
		return testFields{
			config:     cfg,
			count:      &count,
			serverURLs: []string{srv.Listener.Addr().String()},
		}
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
		testFields := setup(t)
		err := Run(context.Background(), "logout")
		assert.NilError(t, err)

		assert.Equal(t, int32(1), atomic.LoadInt32(testFields.count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)

		expected := testFields.config
		expected.Hosts[0].AccessKey = ""
		expected.Hosts[0].Name = ""
		expected.Hosts[0].PolymorphicID = ""
		assert.DeepEqual(t, &expected, updatedCfg)

		updatedKubeCfg, err := clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.DeepEqual(t, expectedKubeCfg, updatedKubeCfg, cmpopts.EquateEmpty())
	})

	t.Run("with clear", func(t *testing.T) {
		testFields := setup(t)
		err := Run(context.Background(), "logout", "--clear")
		assert.NilError(t, err)

		assert.Equal(t, int32(1), atomic.LoadInt32(testFields.count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)

		assert.Equal(t, int32(1), int32(len(updatedCfg.Hosts)))
		assert.DeepEqual(t, testFields.config.Hosts[1], updatedCfg.Hosts[0])
		// assert.DeepEqual(t, &expected, updatedCfg)

		updatedKubeCfg, err := clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.DeepEqual(t, expectedKubeCfg, updatedKubeCfg, cmpopts.EquateEmpty())
	})

	t.Run("with all", func(t *testing.T) {
		testFields := setup(t)
		err := Run(context.Background(), "logout", "--all")
		assert.NilError(t, err)

		assert.Equal(t, int32(2), atomic.LoadInt32(testFields.count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)

		expected := testFields.config
		expected.Hosts[0].AccessKey = ""
		expected.Hosts[0].Name = ""
		expected.Hosts[0].PolymorphicID = ""
		expected.Hosts[1].AccessKey = ""
		expected.Hosts[1].Name = ""
		expected.Hosts[1].PolymorphicID = ""
		assert.DeepEqual(t, &expected, updatedCfg)

		updatedKubeCfg, err := clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.DeepEqual(t, expectedKubeCfg, updatedKubeCfg, cmpopts.EquateEmpty())
	})

	t.Run("with clear all", func(t *testing.T) {
		testFields := setup(t)
		err := Run(context.Background(), "logout", "--clear", "--all")
		assert.NilError(t, err)

		assert.Equal(t, int32(2), atomic.LoadInt32(testFields.count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)

		expected := ClientConfig{Version: "0.3"}
		assert.DeepEqual(t, &expected, updatedCfg)

		updatedKubeCfg, err := clientConfig().RawConfig()
		assert.NilError(t, err)
		assert.DeepEqual(t, expectedKubeCfg, updatedKubeCfg, cmpopts.EquateEmpty())
	})

	t.Run("with one and all", func(t *testing.T) {
		testFields := setup(t)
		err := Run(context.Background(), "logout", testFields.serverURLs[0], "--all")
		assert.Error(t, err, "Argument [SERVER] and flag [--all] cannot be both specified.")

		assert.Equal(t, int32(0), atomic.LoadInt32(testFields.count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)
		assert.DeepEqual(t, &testFields.config, updatedCfg)
	})

	t.Run("error", func(t *testing.T) {
		testFields := setupError(t)
		err := Run(context.Background(), "logout", testFields.serverURLs[0])
		assert.ErrorContains(t, err, "Failed to logout of server "+testFields.serverURLs[0])

		assert.Equal(t, int32(1), atomic.LoadInt32(testFields.count), "calls to API")

		updatedCfg, err := readConfig()
		assert.NilError(t, err)

		expected := testFields.config
		assert.DeepEqual(t, &expected, updatedCfg)
	})
}
