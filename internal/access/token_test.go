package access

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/authn"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/secrets"
)

// mockOIDC is a mock oidc identity provider
type mockOIDCImplementation struct {
	UserEmailResp  string
	UserGroupsResp []string
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

func TestExchangeAuthCodeForProviderTokens(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"NewUserNewGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				return &mockOIDCImplementation{
					UserEmailResp:  "newusernewgroups@example.com",
					UserGroupsResp: []string{"Everyone", "developers"},
				}
			},
			"verify": func(t *testing.T, user *models.User, sessToken string, err error) {
				require.NoError(t, err)
				require.Equal(t, "newusernewgroups@example.com", user.Email)
				require.NotEmpty(t, sessToken)

				require.Len(t, user.Groups, 2)

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				require.Contains(t, groupNames, "Everyone")
				require.Contains(t, groupNames, "developers")
			},
		},
		"NewUserExistingGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				existingGroup1 := &models.Group{Name: "existing1"}
				existingGroup2 := &models.Group{Name: "existing2"}

				_, err := data.CreateGroup(db, existingGroup1)
				require.NoError(t, err)

				_, err = data.CreateGroup(db, existingGroup2)
				require.NoError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "newuserexistinggroups@example.com",
					UserGroupsResp: []string{"existing1", "existing2"},
				}
			},
			"verify": func(t *testing.T, user *models.User, sessToken string, err error) {
				require.NoError(t, err)
				require.Equal(t, "newuserexistinggroups@example.com", user.Email)
				require.NotEmpty(t, sessToken)

				require.Len(t, user.Groups, 2)

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				require.Contains(t, groupNames, "existing1")
				require.Contains(t, groupNames, "existing2")
			},
		},
		"ExistingUserNewGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				_, err := data.CreateUser(db, &models.User{Email: "existingusernewgroups@example.com"})
				require.NoError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "existingusernewgroups@example.com",
					UserGroupsResp: []string{"existingusernewgroups1", "existingusernewgroups2"},
				}
			},
			"verify": func(t *testing.T, user *models.User, sessToken string, err error) {
				require.NoError(t, err)
				require.Equal(t, "existingusernewgroups@example.com", user.Email)
				require.NotEmpty(t, sessToken)

				require.Len(t, user.Groups, 2)

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				require.Contains(t, groupNames, "existingusernewgroups1")
				require.Contains(t, groupNames, "existingusernewgroups2")
			},
		},
		"ExistingUserExistingGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				_, err := data.CreateUser(db, &models.User{Email: "existinguserexistinggroups@example.com"})
				require.NoError(t, err)

				_, err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups1"})
				require.NoError(t, err)

				_, err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups2"})
				require.NoError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "existinguserexistinggroups@example.com",
					UserGroupsResp: []string{"existinguserexistinggroups1", "existinguserexistinggroups2"},
				}
			},
			"verify": func(t *testing.T, user *models.User, sessToken string, err error) {
				require.NoError(t, err)
				require.Equal(t, "existinguserexistinggroups@example.com", user.Email)
				require.NotEmpty(t, sessToken)

				require.Len(t, user.Groups, 2)

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				require.Contains(t, groupNames, "existinguserexistinggroups1")
				require.Contains(t, groupNames, "existinguserexistinggroups2")
			},
		},
	}

	for k, v := range cases {
		// setup db and context
		db := setupDB(t)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("db", db)

		// secret provider setup
		sp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
			Path: os.TempDir(),
		})

		rootKey := "db_at_rest"
		symmetricKeyProvider := secrets.NewNativeSecretProvider(sp)
		symmetricKey, err := symmetricKeyProvider.GenerateDataKey(rootKey)
		require.NoError(t, err)

		models.SymmetricKey = symmetricKey

		// setup fake identity provider
		provider := &models.Provider{Kind: "okta", Domain: "mockOIDC.example.com"}
		provider, err = data.CreateProvider(db, provider)
		require.NoError(t, err)

		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gorm.DB) authn.OIDC)
			require.True(t, ok)
			mockOIDC := setupFunc(t, db)

			u, sess, err := ExchangeAuthCodeForSessionToken(c, "123somecode", provider, mockOIDC, time.Minute)

			verifyFunc, ok := v["verify"].(func(*testing.T, *models.User, string, error))
			require.True(t, ok)

			verifyFunc(t, u, sess, err)
		})
	}
}
