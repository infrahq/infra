package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/poll"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/connector"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestConnector_Run_Kubernetes(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for short run")
	}

	assert.NilError(t, logging.SetLevel("debug"))
	t.Cleanup(func() {
		assert.NilError(t, logging.SetLevel("info"))
	})

	dir := t.TempDir()
	serverOpts := defaultServerOptions(dir)
	setupServerOptions(t, &serverOpts)
	serverOpts.Config = server.Config{
		Users: []server.User{
			{Name: "admin@example.com", AccessKey: "0000000001.adminadminadminadmin1234"},
			{Name: "connector", AccessKey: "0000000002.connectorconnectorconnec"},
		},
		Grants: []server.Grant{
			{User: "user1@example.com", Resource: "testing.ns1", Role: "admin"},
			{User: "user2@example.com", Resource: "testing", Role: "view"},
			{Group: "group1@example.com", Resource: "testing.ns1", Role: "logs"},
		},
	}

	srv, err := server.New(serverOpts)
	assert.NilError(t, err)

	fakeKube := &fakeKubeAPI{t: t}
	kubeSrv := httptest.NewTLSServer(fakeKube)
	t.Cleanup(kubeSrv.Close)

	ctx := context.Background()
	runAndWait(ctx, t, srv.Run)

	kubeconfig := path.Join(dir, "kubeconfig")
	os.Setenv("KUBECONFIG", kubeconfig)
	err = clientcmd.WriteToFile(api.Config{
		Clusters:       map[string]*api.Cluster{"test": {Server: kubeSrv.URL, CertificateAuthorityData: certs.PEMEncodeCertificate(kubeSrv.Certificate().Raw)}},
		Contexts:       map[string]*api.Context{"test": {Cluster: "test", AuthInfo: "test"}},
		AuthInfos:      map[string]*api.AuthInfo{"test": {Token: "auth-token"}},
		CurrentContext: "test",
	}, kubeconfig)

	opts := connector.Options{
		Server: connector.ServerOptions{
			URL:                urlFromAddr(t, srv.Addrs.HTTPS),
			AccessKey:          "0000000002.connectorconnectorconnec",
			TrustedCertificate: serverOpts.TLS.Certificate,
		},
		Name:         "testing",
		Kind:         "kubernetes",
		CACert:       types.StringOrFile(readFile(t, "testdata/pki/connector.crt")),
		CAKey:        types.StringOrFile(readFile(t, "testdata/pki/connector.key")),
		EndpointAddr: types.HostPort{Host: "127.0.0.1", Port: 55555},
		Addr: connector.ListenerOptions{
			HTTP:    "127.0.0.1:0",
			HTTPS:   "127.0.0.1:0",
			Metrics: "127.0.0.1:0",
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(cancel)

	runAndWait(ctx, t, func(ctx context.Context) error {
		return connector.Run(ctx, opts)
	})

	// check destination has been registered
	var destination *models.Destination
	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		destination, err = data.GetDestination(srv.DB(),
			data.GetDestinationOptions{ByName: "testing"})
		switch {
		case errors.Is(err, internal.ErrNotFound):
			return poll.Continue("destination not registered")
		case err != nil:
			return poll.Error(err)
		}
		return poll.Success()
	}, poll.WithTimeout(30*time.Second))

	// check the destination was updated
	expected := &models.Destination{
		Model: models.Model{
			ID:        anyUID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		OrganizationMember: models.OrganizationMember{OrganizationID: srv.DB().OrganizationID()},
		Name:               "testing",
		Kind:               "kubernetes",
		UniqueID:           "4ebfd7dabeec5b37eafd20e3775f70ab86c7422036367d77d9bebfa03864e08b",
		ConnectionURL:      "127.0.0.1:55555",
		ConnectionCA:       opts.CACert.String(),
		LastSeenAt:         time.Now(),
		Version:            "99.99.99999",
		Resources:          models.CommaSeparatedStrings{"default", "ns1", "ns2"},
		Roles:              models.CommaSeparatedStrings{"admin", "view", "edit", "custom", "logs"},
	}
	assert.DeepEqual(t, destination, expected, cmpDestinationModel)

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		fakeKube.writesLock.Lock()
		defer fakeKube.writesLock.Unlock()
		if len(fakeKube.writes) >= 3 {
			return poll.Success()
		}
		return poll.Continue("request count %d waiting for 3", len(fakeKube.writes))
	})

	// check kube bindings were updated
	expectedWrites := []kubeRequest{
		{
			Method: "PUT",
			Path:   "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings/infra:view",
			Body: kubeBindingRequestBody{
				Kind:     "ClusterRoleBinding",
				Metadata: metav1.ObjectMeta{Name: "infra:view"},
				RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: "view"},
				Subjects: []rbacv1.Subject{{Kind: "User", Name: "user2@example.com"}},
			},
		},
		{
			Method: "PUT",
			Path:   "/apis/rbac.authorization.k8s.io/v1/namespaces/ns1/rolebindings/infra:admin",
			Body: kubeBindingRequestBody{
				Kind:     "RoleBinding",
				Metadata: metav1.ObjectMeta{Name: "infra:admin", Namespace: "ns1"},
				RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: "admin"},
				Subjects: []rbacv1.Subject{{Kind: "User", Name: "user1@example.com"}},
			},
		},
		{
			Method: "PUT",
			Path:   "/apis/rbac.authorization.k8s.io/v1/namespaces/ns1/rolebindings/infra:logs",
			Body: kubeBindingRequestBody{
				Kind:     "RoleBinding",
				Metadata: metav1.ObjectMeta{Name: "infra:logs", Namespace: "ns1"},
				RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: "logs"},
				Subjects: []rbacv1.Subject{{Kind: "Group", Name: "group1@example.com"}},
			},
		},
	}
	sort.Slice(fakeKube.writes, func(i, j int) bool {
		return fakeKube.writes[i].Path < fakeKube.writes[j].Path
	})
	assert.DeepEqual(t, fakeKube.writes, expectedWrites, cmpKubeRequest)

	// TODO: check proxy is listening
}

func urlFromAddr(t *testing.T, addr net.Addr) types.URL {
	t.Helper()
	var u types.URL
	assert.NilError(t, u.Set(addr.String()))
	return u
}

var cmpDestinationModel = cmp.Options{
	cmp.FilterPath(opt.PathField(models.Model{}, "ID"), cmpIDNotZero),
	cmp.FilterPath(opt.PathField(models.Model{}, "CreatedAt"),
		opt.TimeWithThreshold(5*time.Second)),
	cmp.FilterPath(opt.PathField(models.Model{}, "UpdatedAt"),
		opt.TimeWithThreshold(5*time.Second)),
	cmp.FilterPath(opt.PathField(models.Destination{}, "LastSeenAt"),
		opt.TimeWithThreshold(5*time.Second)),
}

var cmpKubeRequest = cmp.Options{
	cmpopts.EquateEmpty(),
	cmpopts.IgnoreFields(metav1.ObjectMeta{}, "Labels"),
	cmpopts.IgnoreFields(rbacv1.RoleRef{}, "APIGroup"),
	cmpopts.IgnoreFields(rbacv1.Subject{}, "APIGroup"),
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	raw, err := os.ReadFile(p)
	assert.NilError(t, err)
	return string(raw)
}

type fakeKubeAPI struct {
	t          *testing.T
	writes     []kubeRequest
	writesLock sync.Mutex
}

func (f *fakeKubeAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		f.handleGET(w, req)
	case http.MethodPut:
		f.handlePUT(w, req)
	default:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "unexpected request to fakeKube: %v %v", req.Method, req.URL)
	}
}

func (f *fakeKubeAPI) handleGET(w http.ResponseWriter, req *http.Request) {
	headers := w.Header()
	switch {
	case req.URL.Path == "/apis/rbac.authorization.k8s.io/v1/clusterroles":
		roleMap := map[string][]string{
			"kubernetes.io/bootstrapping=rbac-defaults": {"admin", "view", "edit"},
			"app.infrahq.com/include-role=true":         {"custom", "logs"},
		}

		headers.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		selector := req.URL.Query().Get("labelSelector")
		roles := roleMap[selector]

		if selector == "" {
			for _, items := range roleMap {
				roles = append(roles, items...)
			}
		}

		result := rbacv1.ClusterRoleList{}
		for _, role := range roles {
			result.Items = append(result.Items,
				rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: role}})
		}
		assert.Check(f.t, json.NewEncoder(w).Encode(result))

	case req.URL.Path == "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings":
		headers.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		result := rbacv1.ClusterRoleBindingList{
			Items: []rbacv1.ClusterRoleBinding{},
		}
		assert.Check(f.t, json.NewEncoder(w).Encode(result))

	case req.URL.Path == "/apis/rbac.authorization.k8s.io/v1/rolebindings":
		headers.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		result := rbacv1.RoleBindingList{
			Items: []rbacv1.RoleBinding{},
		}
		assert.Check(f.t, json.NewEncoder(w).Encode(result))

	case req.URL.Path == "/api/v1/namespaces":
		headers.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		result := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ns1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ns2"}},
			},
		}
		assert.Check(f.t, json.NewEncoder(w).Encode(result))
	default:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "unexpected request to fakeKube: %v %v", req.Method, req.URL)
	}
}

func (f *fakeKubeAPI) handlePUT(w http.ResponseWriter, req *http.Request) {
	f.writesLock.Lock()
	defer f.writesLock.Unlock()

	kubeReq := kubeRequest{
		Method: req.Method,
		Path:   req.URL.Path,
		Query:  req.URL.Query(),
	}
	assert.NilError(f.t, json.NewDecoder(req.Body).Decode(&kubeReq.Body))
	f.writes = append(f.writes, kubeReq)

	headers := w.Header()
	switch {
	case strings.HasPrefix(req.URL.Path, "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings/"):
		headers.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		result := rbacv1.ClusterRoleBinding{}
		assert.Check(f.t, json.NewEncoder(w).Encode(result))

	case strings.HasPrefix(req.URL.Path, "/apis/rbac.authorization.k8s.io/v1/"):
		headers.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		result := rbacv1.ClusterRoleBinding{}
		assert.Check(f.t, json.NewEncoder(w).Encode(result))

	default:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "unexpected request to fakeKube: %v %v", req.Method, req.URL)
	}
}

type kubeRequest struct {
	Method string
	Path   string
	Query  url.Values
	Body   kubeBindingRequestBody
}

type kubeBindingRequestBody struct {
	Kind     string
	Metadata metav1.ObjectMeta
	RoleRef  rbacv1.RoleRef
	Subjects []rbacv1.Subject
}

func TestConnectorCmd_LoadConfig(t *testing.T) {
	type testCase struct {
		name     string
		config   string
		setup    func(t *testing.T)
		expected func() connector.Options
	}

	run := func(t *testing.T, tc testCase) {
		var actual connector.Options
		patchRunConnector(t, func(ctx context.Context, options connector.Options) error {
			actual = options
			return nil
		})

		if tc.setup != nil {
			tc.setup(t)
		}

		dir := fs.NewDir(t, t.Name(), fs.WithFile("config.yaml", tc.config))

		ctx := context.Background()
		err := Run(ctx, "connector", "-f", dir.Join("config.yaml"))
		assert.NilError(t, err)
		assert.DeepEqual(t, actual, tc.expected())
	}

	keyDir := fs.NewDir(t, t.Name(), fs.WithFile("accesskeyfile", "the-access-key"))
	filename := keyDir.Join("accesskeyfile")

	testCases := []testCase{
		{
			name: "full config",
			config: `
server:
  url: the-server
  accessKey: /var/run/secrets/key
  skipTLSVerify: true
  trustedCertificate: ca.pem
name: the-name
kind: ssh
caCert: /path/to/cert
caKey: /path/to/key
addr:
  http: localhost:84
  https: localhost:414
  metrics: 127.0.0.1:8000

ssh:
  group: the-group
  sshdConfigPath: /opt/sshd
`,
			expected: func() connector.Options {
				return connector.Options{
					Name: "the-name",
					Kind: "ssh",
					Addr: connector.ListenerOptions{
						HTTP:    "localhost:84",
						HTTPS:   "localhost:414",
						Metrics: "127.0.0.1:8000",
					},
					Server: connector.ServerOptions{
						URL:                types.URL{Scheme: "http", Host: "the-server"},
						AccessKey:          "/var/run/secrets/key",
						SkipTLSVerify:      true,
						TrustedCertificate: "ca.pem",
					},
					CACert: "/path/to/cert",
					CAKey:  "/path/to/key",
					SSH: connector.SSHOptions{
						Group:          "the-group",
						SSHDConfigPath: "/opt/sshd",
					},
				}
			},
		},
		{
			name:   "access key with file: prefix (deprecated)",
			config: fmt.Sprintf("server:\n  accessKey: file:%v\n", filename),
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-access-key"
				return expected
			},
		},
		{
			name:   "access key from file",
			config: fmt.Sprintf("server:\n  accessKey: %v\n", filename),
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-access-key"
				return expected
			},
		},
		{
			name: "access key with env: prefix (deprecated)",
			setup: func(t *testing.T) {
				t.Setenv("CUSTOM_ENV_VAR", "the-key-from-env")
			},
			config: `
server:
  accessKey: env:CUSTOM_ENV_VAR
`,
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-key-from-env"
				return expected
			},
		},
		{
			name: "access key from INFRA_ACCESS_KEY",
			setup: func(t *testing.T) {
				t.Setenv("INFRA_ACCESS_KEY", "the-key-from-env")
			},
			config: `{}`,
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-key-from-env"
				return expected
			},
		},
		{
			name: "access key from INFRA_ACCESS_KEY points at a file",
			setup: func(t *testing.T) {
				t.Setenv("INFRA_ACCESS_KEY", filename)
			},
			config: `{}`,
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-access-key"
				return expected
			},
		},
		{
			name:   "access key from INFRA_CONNECTOR_SERVER_ACCESS_KEY",
			config: `{}`,
			setup: func(t *testing.T) {
				t.Setenv("INFRA_CONNECTOR_SERVER_ACCESS_KEY", "the-key-from-env")
			},
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-key-from-env"
				return expected
			},
		},
		{
			name: "access key literal from file",
			config: `
server:
  accessKey: the-literal-key
`,
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-literal-key"
				return expected
			},
		},
		{
			name: "access key literal with plaintext prefix (deprecated)",
			config: `
server:
  accessKey: plaintext:the-literal-key
`,
			expected: func() connector.Options {
				expected := defaultConnectorOptions()
				expected.Server.AccessKey = "the-literal-key"
				return expected
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
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
