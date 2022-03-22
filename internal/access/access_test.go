package access

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
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

var (
	tom       = &models.User{Email: "tom@infrahq.com"}
	tomsGroup = &models.Group{Name: "tom's group"}
)

func TestBasicGrant(t *testing.T) {
	db := setupDB(t)
	err := data.CreateUser(db, tom)
	require.NoError(t, err)

	grant(t, db, tom, "u:steven", "read", "infra.groups.1")
	can(t, db, "u:steven", "read", "infra.groups.1")
	cant(t, db, "u:steven", "read", "infra.groups")
	cant(t, db, "u:steven", "read", "infra.groups.2")
	cant(t, db, "u:steven", "write", "infra.groups.1")

	grant(t, db, tom, "u:bob", "read", "infra.groups")
	can(t, db, "u:bob", "read", "infra.groups")
	cant(t, db, "u:bob", "read", "infra.groups.1") // currently we check for exact grant match, this may change as grants evolve
	cant(t, db, "u:bob", "write", "infra.groups")

	grant(t, db, tom, "u:alice", "read", "infra.machines")
	can(t, db, "u:alice", "read", "infra.machines")
	cant(t, db, "u:alice", "read", "infra")
	cant(t, db, "u:alice", "read", "infra.machines.1")
	cant(t, db, "u:alice", "write", "infra.machines")
}

func TestUsersGroupGrant(t *testing.T) {
	db := setupDB(t)
	err := data.CreateUser(db, tom)
	require.NoError(t, err)

	err = data.CreateGroup(db, tomsGroup)
	require.NoError(t, err)

	err = data.BindGroupUsers(db, tomsGroup, *tom)
	require.NoError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", db)
	c.Set("identity", tom.PolymorphicIdentifier())
	c.Set("user", tom)

	grant(t, db, tom, tomsGroup.PolymorphicIdentifier(), models.InfraUserRole, "infra")

	authDB, err := requireInfraRole(c, models.InfraUserRole)
	assert.NoError(t, err)
	assert.NotNil(t, authDB)

	authDB, err = requireInfraRole(c, models.InfraAdminRole)
	assert.Error(t, err)
	assert.Nil(t, authDB)

	authDB, err = requireInfraRole(c, models.InfraAdminRole, models.InfraUserRole)
	assert.NoError(t, err)
	assert.NotNil(t, authDB)
}

func grant(t *testing.T, db *gorm.DB, currentUser *models.User, identity uid.PolymorphicID, privilege, resource string) {
	err := data.CreateGrant(db, &models.Grant{
		Identity:  identity,
		Privilege: privilege,
		Resource:  resource,
		CreatedBy: currentUser.ID,
	})
	require.NoError(t, err)
}

func can(t *testing.T, db *gorm.DB, identity uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, identity, privilege, resource)
	require.NoError(t, err)
	require.True(t, canAccess)
}

func cant(t *testing.T, db *gorm.DB, identity uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, identity, privilege, resource)
	require.NoError(t, err)
	require.False(t, canAccess)
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
	return &authn.UserInfo{Email: m.UserEmailResp, Groups: &m.UserGroupsResp}, nil
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

		SetupTestSecretProvider(t)

		// setup fake identity provider
		provider := &models.Provider{Name: "mockoidc", URL: "mockOIDC.example.com"}
		err := data.CreateProvider(db, provider)
		require.NoError(t, err)

		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gorm.DB) authn.OIDC)
			require.True(t, ok)
			mockOIDC := setupFunc(t, db)

			u, sess, err := ExchangeAuthCodeForAccessKey(c, "123somecode", provider, mockOIDC, time.Minute, "example.com")

			verifyFunc, ok := v["verify"].(func(*testing.T, *models.User, string, error))
			require.True(t, ok)

			verifyFunc(t, u, sess, err)
		})
	}
}

func SetupTestSecretProvider(t *testing.T) {
	sp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	rootKey := "db_at_rest"
	symmetricKeyProvider := secrets.NewNativeSecretProvider(sp)
	symmetricKey, err := symmetricKeyProvider.GenerateDataKey(rootKey)
	require.NoError(t, err)

	models.SymmetricKey = symmetricKey
}
