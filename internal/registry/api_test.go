// Don't add to this file. Create model-specific files for tests.
package registry

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/registry/authn"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/secrets"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockSecretReader struct{}

func NewMockSecretReader() secrets.SecretStorage {
	return &mockSecretReader{}
}

func (msr *mockSecretReader) GetSecret(secretName string) ([]byte, error) {
	return []byte("abcdefghijklmnopqrstuvwx"), nil
}

func (msr *mockSecretReader) SetSecret(secretName string, secret []byte) error {
	return nil
}

func issueAPIToken(t *testing.T, db *gorm.DB, permissions string) *models.APIToken {
	apiToken := &models.APIToken{
		Name:        "test",
		Permissions: permissions,
		TTL:         1 * time.Hour,
	}

	err := data.CreateAPIToken(db, apiToken)
	require.NoError(t, err)

	tkn := &models.Token{APITokenID: apiToken.ID, SessionDuration: apiToken.TTL}
	err = data.CreateToken(db, tkn)
	require.NoError(t, err)

	return apiToken
}

func TestCreateDestination(t *testing.T) {
	cases := []struct {
		Name        string
		AuthFunc    func(t *testing.T, db *gorm.DB, c *gin.Context)
		RequestFunc func(t *testing.T) *api.CreateDestinationRequest
		VerifyFunc  func(t *testing.T, r *api.Destination, err error)
	}{
		{
			Name: "OK",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T) *api.CreateDestinationRequest {
				return &api.CreateDestinationRequest{
					Kind:   api.DestinationKindKubernetes,
					NodeID: "test",
					Name:   "test",
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *api.Destination, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name: "NoKind",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(access.PermissionDestinationCreate))
			},
			RequestFunc: func(t *testing.T) *api.CreateDestinationRequest {
				return &api.CreateDestinationRequest{
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *api.Destination, err error) {
				require.Error(t, err)
			},
		},
		{
			Name: "NoAuthorization",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", "")
			},
			RequestFunc: func(t *testing.T) *api.CreateDestinationRequest {
				return &api.CreateDestinationRequest{
					Kind: api.DestinationKindKubernetes,
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *api.Destination, err error) {
				require.ErrorIs(t, err, internal.ErrForbidden)
			},
		},
		{
			Name: "BadPermissions",
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", "infra.bad.permissions")
			},
			RequestFunc: func(t *testing.T) *api.CreateDestinationRequest {
				return &api.CreateDestinationRequest{
					Kind: api.DestinationKindKubernetes,
					Kubernetes: &api.DestinationKubernetes{
						CA:       "CA",
						Endpoint: "develop.infrahq.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *api.Destination, err error) {
				require.ErrorIs(t, err, internal.ErrForbidden)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			_, db := configure(t, nil)

			c, _ := gin.CreateTestContext(nil)
			c.Set("db", db)

			test.AuthFunc(t, db, c)

			a := API{
				registry: &Registry{
					config: Config{
						Users:  []ConfigUserMapping{},
						Groups: []ConfigGroupMapping{},
					},
				},
			}

			resp, err := a.CreateDestination(c, test.RequestFunc(t))
			test.VerifyFunc(t, resp, err)
		})
	}
}

func TestLogin(t *testing.T) {
	cases := []struct {
		Name        string
		RequestFunc func(t *testing.T) *api.LoginRequest
		VerifyFunc  func(t *testing.T, r *api.LoginResponse, err error)
	}{
		{
			Name: "NilRequest",
			RequestFunc: func(t *testing.T) *api.LoginRequest {
				return &api.LoginRequest{}
			},
			VerifyFunc: func(t *testing.T, r *api.LoginResponse, err error) {
				require.ErrorIs(t, err, internal.ErrBadRequest)
			},
		},
		{
			Name: "OktaNil",
			RequestFunc: func(t *testing.T) *api.LoginRequest {
				return &api.LoginRequest{
					Okta: nil,
				}
			},
			VerifyFunc: func(t *testing.T, r *api.LoginResponse, err error) {
				require.ErrorIs(t, err, internal.ErrBadRequest)
			},
		},
		{
			Name: "OktaEmpty",
			RequestFunc: func(t *testing.T) *api.LoginRequest {
				return &api.LoginRequest{
					Okta: &api.LoginRequestOkta{},
				}
			},
			VerifyFunc: func(t *testing.T, r *api.LoginResponse, err error) {
				require.Error(t, err)
			},
		},
		{
			Name: "OktaMissingDomain",
			RequestFunc: func(t *testing.T) *api.LoginRequest {
				return &api.LoginRequest{
					Okta: &api.LoginRequestOkta{
						Code: "code",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *api.LoginResponse, err error) {
				require.Error(t, err)
			},
		},
		{
			Name: "OktaMissingCodeRequest",
			RequestFunc: func(t *testing.T) *api.LoginRequest {
				return &api.LoginRequest{
					Okta: &api.LoginRequestOkta{
						Domain: "test.okta.com",
					},
				}
			},
			VerifyFunc: func(t *testing.T, r *api.LoginResponse, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			_, db := configure(t, nil)

			c, _ := gin.CreateTestContext(nil)
			c.Set("db", db)

			a := &API{
				registry: &Registry{secrets: map[string]secrets.SecretStorage{
					"base64": NewMockSecretReader(),
				}},
			}

			resp, err := a.Login(c, test.RequestFunc(t))

			test.VerifyFunc(t, resp, err)
		})
	}
}

type mockOIDCImplementation struct {
	UserEmailResp  string
	UserGroupsResp []string
}

func NewMockOIDC(userEmail string, userGroups []string) authn.OIDC {
	return &mockOIDCImplementation{
		UserEmailResp:  userEmail,
		UserGroupsResp: userGroups,
	}
}

func (m *mockOIDCImplementation) ExchangeAuthCodeForProviderTokens(code string) (acc, ref string, exp time.Time, email string, err error) {
	return "acc", "ref", exp, m.UserEmailResp, nil
}

func (o *mockOIDCImplementation) RefreshAccessToken(providerTokens *models.ProviderToken) (accessToken string, expiry *time.Time, err error) {
	// never update
	return string(providerTokens.AccessToken), &providerTokens.Expiry, nil
}

func (m *mockOIDCImplementation) GetUserInfo(providerTokens *models.ProviderToken) (*authn.UserInfo, error) {
	return &authn.UserInfo{Email: m.UserEmailResp, Groups: m.UserGroupsResp}, nil
}

func TestLoginOkta(t *testing.T) {
	_, db := configure(t, nil)

	request := api.LoginRequest{
		Okta: &api.LoginRequestOkta{
			Domain: "test.okta.com",
			Code:   "code",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	testOIDC := NewMockOIDC("jbond@infrahq.com", []string{})
	c.Set("oidc", testOIDC)
	c.Set("db", db)

	a := API{
		registry: &Registry{
			secrets: map[string]secrets.SecretStorage{
				"base64": NewMockSecretReader(),
			},
			options: Options{SessionDuration: oneHundredYears},
		},
	}

	resp, err := a.Login(c, &request)
	require.NoError(t, err)

	require.Equal(t, "jbond@infrahq.com", resp.Name)
	require.NotEmpty(t, resp.Token)
}

func TestCreateAPIToken(t *testing.T) {
	_, db := configure(t, nil)

	request := &api.InfraAPITokenCreateRequest{
		Name:        "tmp",
		Permissions: []string{string(access.PermissionAllInfra)},
	}

	c, _ := gin.CreateTestContext(nil)
	c.Set("db", db)
	c.Set("permissions", string(access.PermissionAllInfra))

	a := API{}

	_, err := a.CreateAPIToken(c, request)

	require.NoError(t, err)

	newc, _ := gin.CreateTestContext(nil)
	newc.Set("db", db)
	newc.Set("permissions", string(access.PermissionUserRead))

	_, err = a.ListUsers(newc, &api.ListUsersRequest{})
	require.NoError(t, err)
}

func TestDeleteAPIToken(t *testing.T) {
	_, db := configure(t, nil)

	permissions := strings.Join([]string{
		string(access.PermissionUserRead),
		string(access.PermissionAPITokenDelete),
	}, " ")

	apiToken := issueAPIToken(t, db, permissions)

	oldc, _ := gin.CreateTestContext(nil)
	oldc.Set("db", db)
	oldc.Set("permissions", permissions)

	a := API{}

	_, err := a.ListUsers(oldc, &api.ListUsersRequest{})

	require.NoError(t, err)

	c, _ := gin.CreateTestContext(nil)
	c.Set("db", db)
	c.Set("permissions", permissions)

	err = a.DeleteAPIToken(c, &api.Resource{ID: apiToken.ID})
	require.NoError(t, err)
}
