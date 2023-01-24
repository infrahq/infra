package connector

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gotest.tools/v3/assert"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/uid"
)

func TestAuthenticator_Authenticate(t *testing.T) {
	type testCase struct {
		name        string
		setup       func(t *testing.T, req *http.Request)
		fakeClient  fakeClient
		expectedErr string
		expected    func(t *testing.T, claims claims.Custom)
	}

	pub, priv := generateJWK(t)

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, "/apis", nil)
		if tc.setup != nil {
			tc.setup(t, req)
		}

		opts := Options{
			Server: ServerOptions{SkipTLSVerify: true, AccessKey: "the-access-key"},
		}
		assert.NilError(t, opts.Server.URL.Set("https://127.0.0.1:12345"))
		authn := newAuthenticator(opts)
		authn.client = tc.fakeClient

		actual, err := authn.Authenticate(req)
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr)
			return
		}
		assert.NilError(t, err)

		if tc.expected != nil {
			tc.expected(t, actual)
		}
	}

	testCases := []testCase{
		{
			name:        "no auth header",
			expectedErr: "no bearer token found",
		},
		{
			name: "no token",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "username:password")
			},
			expectedErr: "invalid JWT signature",
		},
		{
			name: "invalid JWK",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")
			},
			fakeClient:  fakeClient{err: fmt.Errorf("server not available")},
			expectedErr: "get JWK from server: server not available",
		},
		{
			name: "invalid JWT",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")
			},
			fakeClient:  fakeClient{key: *pub},
			expectedErr: "error in cryptographic primitive",
		},
		{
			name: "expired JWT",
			setup: func(t *testing.T, req *http.Request) {
				j := generateJWT(t, priv, "test@example.com", time.Now().Add(-1*time.Hour))
				req.Header.Set("Authorization", "Bearer "+j)
			},
			fakeClient:  fakeClient{key: *pub},
			expectedErr: "token is expired",
		},
		{
			name: "no username in JWT",
			setup: func(t *testing.T, req *http.Request) {
				j := generateJWT(t, priv, "", time.Now().Add(time.Hour))
				req.Header.Set("Authorization", "Bearer "+j)
			},
			fakeClient:  fakeClient{key: *pub},
			expectedErr: "no username in JWT claim",
		},
		{
			name: "valid JWT",
			setup: func(t *testing.T, req *http.Request) {
				j := generateJWT(t, priv, "test@example.com", time.Now().Add(time.Hour))
				req.Header.Set("Authorization", "Bearer "+j)
			},
			fakeClient: fakeClient{key: *pub},
			expected: func(t *testing.T, actual claims.Custom) {
				expected := claims.Custom{
					Name:   "test@example.com",
					Groups: []string{"developers"},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
		{
			name: "error status code from server",
			setup: func(t *testing.T, req *http.Request) {
				j := generateJWT(t, priv, "test@example.com", time.Now().Add(time.Hour))
				req.Header.Set("Authorization", "Bearer "+j)
			},
			fakeClient:  fakeClient{key: *pub, statusCode: http.StatusBadRequest},
			expectedErr: "Bad Request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func generateJWK(t *testing.T) (pub *jose.JSONWebKey, priv *jose.JSONWebKey) {
	t.Helper()
	pubkey, key, err := ed25519.GenerateKey(rand.Reader)
	assert.NilError(t, err)

	priv = &jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.ED25519), Use: "sig"}
	thumb, err := priv.Thumbprint(crypto.SHA256)
	assert.NilError(t, err)

	kid := base64.URLEncoding.EncodeToString(thumb)
	priv.KeyID = kid
	pub = &jose.JSONWebKey{Key: pubkey, KeyID: kid, Algorithm: string(jose.ED25519), Use: "sig"}
	return pub, priv
}

func generateJWT(t *testing.T, priv *jose.JSONWebKey, email string, expiry time.Time) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: priv}, (&jose.SignerOptions{}).WithType("JWT"))
	assert.NilError(t, err)

	cl := jwt.Claims{
		Issuer:   "InfraHQ",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	custom := claims.Custom{
		Name:   email,
		Groups: []string{"developers"},
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	assert.NilError(t, err)
	return raw
}

type fakeClient struct {
	key        jose.JSONWebKey
	err        error
	statusCode int
}

func (f fakeClient) Do(req *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}

	r := server.WellKnownJWKResponse{Keys: []jose.JSONWebKey{f.key}}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(r)
	if err != nil {
		return nil, err
	}

	if authHeader := req.Header.Get("Authorization"); authHeader != "Bearer the-access-key" {
		return nil, fmt.Errorf("missing authorization header or wrong access key: %v", authHeader)
	}

	code := http.StatusOK
	if f.statusCode != 0 {
		code = f.statusCode
	}
	resp := &http.Response{
		StatusCode: code,
		Status:     http.StatusText(code),
		Body:       io.NopCloser(&buf),
	}
	return resp, nil
}

func TestCertCache_Certificate(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for short run")
	}
	testCACertPEM, err := os.ReadFile("./_testdata/test-ca-cert.pem")
	assert.NilError(t, err)

	testCAKeyPEM, err := os.ReadFile("./_testdata/test-ca-key.pem")
	assert.NilError(t, err)

	t.Run("no cached certificate adds empty certificate", func(t *testing.T) {
		certCache := NewCertCache(testCACertPEM, testCAKeyPEM)

		cert, err := certCache.Certificate()

		assert.NilError(t, err)
		assert.Equal(t, len(certCache.hosts), 1)
		assert.Equal(t, certCache.hosts[0], "")
		assert.Assert(t, cert != nil)
	})

	t.Run("cached certificate is returned when the host is set", func(t *testing.T) {
		certCache := NewCertCache(testCACertPEM, testCAKeyPEM)
		_, err := certCache.AddHost("test-host")
		assert.NilError(t, err)

		cert, err := certCache.Certificate()

		assert.NilError(t, err)
		assert.Equal(t, len(certCache.hosts), 1)
		assert.Equal(t, certCache.hosts[0], "test-host")

		parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
		assert.NilError(t, err)
		assert.Equal(t, len(parsedCert.DNSNames), 1)
		assert.Equal(t, parsedCert.DNSNames[0], "test-host")
	})
}

func TestSyncGrantsToDestination_KubeBindings(t *testing.T) {
	type testCase struct {
		name                     string
		fakeAPI                  *fakeAPIClient
		fakeKube                 *fakeKubeClient
		expectedListGrantIndexes []int64
		successCount             int
	}

	run := func(t *testing.T, tc testCase) {
		ctx := context.Background()
		waiter := &fakeWaiter{endAtIndex: 1}
		if tc.fakeKube == nil {
			tc.fakeKube = &fakeKubeClient{}
		}
		con := connector{
			k8s:         tc.fakeKube,
			client:      tc.fakeAPI,
			destination: &api.Destination{Name: "the-dest"},
		}

		fn := func(ctx context.Context, grants []api.Grant) error {
			return updateRoles(ctx, con.client, con.k8s, grants)
		}
		err := syncGrantsToDestination(ctx, con, waiter, fn)
		assert.ErrorIs(t, err, errDone)

		assert.Equal(t, len(waiter.resets), tc.successCount)
		assert.DeepEqual(t, tc.fakeAPI.listGrantsIndexes, tc.expectedListGrantIndexes)
	}

	testCases := []testCase{
		{
			name: "successful update",
			fakeAPI: &fakeAPIClient{
				listGrantsResult: &api.ListResponse[api.Grant]{
					Items: []api.Grant{
						{User: uid.ID(123), DestinationName: "the-test", Privilege: "view"},
						{User: uid.ID(124), DestinationName: "the-test", DestinationResource: "ns1", Privilege: "logs"},
					},
					LastUpdateIndex: api.LastUpdateIndex{Index: 42},
				},
			},
			expectedListGrantIndexes: []int64{1, 42},
			successCount:             2,
		},
		{
			name: "api blocking request timeout",
			fakeAPI: &fakeAPIClient{
				listGrantsError: api.Error{Code: http.StatusNotModified},
			},
			expectedListGrantIndexes: []int64{1, 1},
			successCount:             2,
		},
		{
			name: "error from api",
			fakeAPI: &fakeAPIClient{
				listGrantsError: api.Error{Code: http.StatusInternalServerError},
			},
			expectedListGrantIndexes: []int64{1, 1},
		},
		{
			name: "failed to update kube",
			fakeAPI: &fakeAPIClient{
				listGrantsResult: &api.ListResponse[api.Grant]{
					Items: []api.Grant{
						{User: uid.ID(123), DestinationName: "the-test", Privilege: "view"},
					},
					LastUpdateIndex: api.LastUpdateIndex{Index: 42},
				},
			},
			expectedListGrantIndexes: []int64{1, 1},
			fakeKube: &fakeKubeClient{
				updateBindingsError: fmt.Errorf("failed to update"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

type fakeWaiter struct {
	index      int
	resets     []int
	endAtIndex int
}

func (f *fakeWaiter) Reset() {
	f.resets = append(f.resets, f.index)
}

func (f *fakeWaiter) Wait(ctx context.Context) error {
	if f.index >= f.endAtIndex {
		return errDone
	}
	if ctx.Err() != nil {
		return errDone
	}
	f.index++
	return nil
}

var errDone = fmt.Errorf("done")

type fakeAPIClient struct {
	api.Client

	listGrantsResult  *api.ListResponse[api.Grant]
	listGrantsError   error
	listGrantsIndexes []int64

	users map[uid.ID]api.User
}

func (f *fakeAPIClient) ListGrants(ctx context.Context, req api.ListGrantsRequest) (*api.ListResponse[api.Grant], error) {
	f.listGrantsIndexes = append(f.listGrantsIndexes, req.LastUpdateIndex)
	return f.listGrantsResult, f.listGrantsError
}

func (f *fakeAPIClient) GetGroup(ctx context.Context, id uid.ID) (*api.Group, error) {
	return &api.Group{Name: "the-group"}, nil
}

func (f *fakeAPIClient) GetUser(ctx context.Context, id uid.ID) (*api.User, error) {
	if user, ok := f.users[id]; ok {
		return &user, nil
	}
	return &api.User{Name: "theuser@example.com"}, nil
}

type fakeKubeClient struct {
	kubernetes.Kubernetes
	updateBindingsError           error
	updateClusterRoleBindingsArgs []map[string][]rbacv1.Subject
	updateRoleBindingsArgs        []map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject
}

func (f *fakeKubeClient) UpdateClusterRoleBindings(subjects map[string][]rbacv1.Subject) error {
	f.updateClusterRoleBindingsArgs = append(f.updateClusterRoleBindingsArgs, subjects)
	return f.updateBindingsError
}

func (f *fakeKubeClient) UpdateRoleBindings(subjects map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) error {
	f.updateRoleBindingsArgs = append(f.updateRoleBindingsArgs, subjects)
	return f.updateBindingsError
}
