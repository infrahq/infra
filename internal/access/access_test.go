package access

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/authn"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	return db
}

func TestRequireAuthorization(t *testing.T) {
	cases := []struct {
		Name                string
		RequiredPermissions []Permission
		AuthFunc            func(t *testing.T, db *gorm.DB, c *gin.Context)
		VerifyFunc          func(t *testing.T, err error)
	}{
		{
			Name:                "AuthorizedAll",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAll))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedAllInfra",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAllInfra))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedExactMatch",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionUserRead))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedOneOfMany",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionUserRead), string(PermissionUserCreate), string(PermissionUserDelete)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedWildcardAction",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionUser))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "AuthorizedWildcardResource",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAllRead))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "APITokenAuthorizedNotFirst",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionGroupRead), string(PermissionProviderRead), string(PermissionUserRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "APITokenAuthorizedNotLast",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionGroupRead), string(PermissionUserRead), string(PermissionProviderRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			Name:                "APITokenAuthorizedNoMatch",
			RequiredPermissions: []Permission{PermissionUserRead},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionUserCreate), string(PermissionGroupRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.EqualError(t, err, `missing permission "infra.user.read": forbidden`)
			},
		},
		{
			Name:                "NotRequired",
			RequiredPermissions: []Permission{},
			AuthFunc: func(t *testing.T, db *gorm.DB, c *gin.Context) {
			},
			VerifyFunc: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			db := setupDB(t)

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("db", db)

			test.AuthFunc(t, db, c)

			_, err := requireAuthorization(c, test.RequiredPermissions...)
			test.VerifyFunc(t, err)
		})
	}
}

func TestRequireAuthorizationWithCheck(t *testing.T) {
	userID := uid.New()

	tests := []struct {
		Name             string
		CurrentUser      *models.User
		PermissionWanted Permission
		CustomCheckFn    func(cUser *models.User) bool
		ErrorExpected    error
	}{
		{
			Name:             "wrong user with no permissions",
			CurrentUser:      &models.User{},
			PermissionWanted: PermissionDestinationRead,
			CustomCheckFn:    func(cUser *models.User) bool { return false },
			ErrorExpected:    internal.ErrForbidden,
		},
		{
			Name:             "right user with no permissions",
			CurrentUser:      &models.User{Model: models.Model{ID: userID}},
			PermissionWanted: PermissionDestinationRead,
			CustomCheckFn:    func(cUser *models.User) bool { return cUser.ID == userID },
			ErrorExpected:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			db := setupDB(t)
			c, _ := gin.CreateTestContext(nil)
			c.Set("db", db)
			c.Set("permissions", test.CurrentUser.Permissions)
			c.Set("user", test.CurrentUser)

			_, err := requireAuthorizationWithCheck(c, test.PermissionWanted, test.CustomCheckFn)

			if test.ErrorExpected != nil {
				require.ErrorIs(t, err, test.ErrorExpected)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAllRequired(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"ExactMatch": {
			"permissions": []string{string(PermissionUserRead)},
			"required":    []string{string(PermissionUserRead)},
			"expected":    true,
		},
		"SubsetMatch": {
			"permissions": []string{string(PermissionUserCreate), string(PermissionUserRead)},
			"required":    []string{string(PermissionUserRead)},
			"expected":    true,
		},
		"NoMatch": {
			"permissions": []string{string(PermissionUserCreate), string(PermissionUserRead)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    false,
		},
		"NoPermissions": {
			"permissions": []string{},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    false,
		},
		"AllPermissions": {
			"permissions": []string{string(PermissionAll)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
		"AllPermissionsAlternate": {
			"permissions": []string{string(PermissionAllInfra)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
		"AllPermissionsForResource": {
			"permissions": []string{string(PermissionUser)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			permissions, ok := v["permissions"].([]string)
			require.True(t, ok)

			required, ok := v["required"].([]string)
			require.True(t, ok)

			result := AllRequired(permissions, required)

			expected, ok := v["expected"].(bool)
			require.True(t, ok)

			assert.Equal(t, expected, result)
		})
	}
}

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
	return string(providerTokens.AccessToken), &providerTokens.ExpiresAt, nil
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
			},
		},
		"NewUserExistingGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				existingGroup1 := &models.Group{Name: "existing1"}
				existingGroup2 := &models.Group{Name: "existing2"}

				err := data.CreateGroup(db, existingGroup1)
				require.NoError(t, err)

				err = data.CreateGroup(db, existingGroup2)
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
				err := data.CreateUser(db, &models.User{Email: "existingusernewgroups@example.com"})
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
				err := data.CreateUser(db, &models.User{Email: "existinguserexistinggroups@example.com"})
				require.NoError(t, err)

				err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups1"})
				require.NoError(t, err)

				err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups2"})
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
		provider := &models.Provider{Name: "mockoidc", URL: "mockOIDC.example.com"}
		err = data.CreateProvider(db, provider)
		require.NoError(t, err)

		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gorm.DB) authn.OIDC)
			require.True(t, ok)
			mockOIDC := setupFunc(t, db)

			u, sess, err := ExchangeAuthCodeForAPIToken(c, "123somecode", provider, mockOIDC, time.Minute)

			verifyFunc, ok := v["verify"].(func(*testing.T, *models.User, string, error))
			require.True(t, ok)

			verifyFunc(t, u, sess, err)
		})
	}
}
