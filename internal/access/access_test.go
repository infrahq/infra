package access

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := data.NewDB(driver)
	assert.NilError(t, err)

	return db
}

func setupAccessTestContext(t *testing.T) (*gin.Context, *gorm.DB, *models.Provider) {
	// setup db and context
	db := setupDB(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", db)

	admin := &models.Identity{Name: "admin@example.com", Kind: models.UserKind}
	err := data.CreateIdentity(db, admin)
	assert.NilError(t, err)

	c.Set("identity", admin)

	adminGrant := &models.Grant{
		Subject:   admin.PolyID(),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
	}
	err = data.CreateGrant(db, adminGrant)
	assert.NilError(t, err)

	SetupTestSecretProvider(t)

	provider := &models.Provider{Name: models.InternalInfraProviderName}
	err = data.CreateProvider(db, provider)
	assert.NilError(t, err)

	return c, db, provider
}

var (
	tom       = &models.Identity{Name: "tom@infrahq.com", Kind: models.UserKind}
	tomsGroup = &models.Group{Name: "tom's group"}
)

func TestBasicGrant(t *testing.T) {
	db := setupDB(t)
	err := data.CreateIdentity(db, tom)
	assert.NilError(t, err)

	grant(t, db, tom, "i:steven", "read", "infra.groups.1")
	can(t, db, "i:steven", "read", "infra.groups.1")
	cant(t, db, "i:steven", "read", "infra.groups")
	cant(t, db, "i:steven", "read", "infra.groups.2")
	cant(t, db, "i:steven", "write", "infra.groups.1")

	grant(t, db, tom, "i:bob", "read", "infra.groups")
	can(t, db, "i:bob", "read", "infra.groups")
	cant(t, db, "i:bob", "read", "infra.groups.1") // currently we check for exact grant match, this may change as grants evolve
	cant(t, db, "i:bob", "write", "infra.groups")

	grant(t, db, tom, "i:alice", "read", "infra.machines")
	can(t, db, "i:alice", "read", "infra.machines")
	cant(t, db, "i:alice", "read", "infra")
	cant(t, db, "i:alice", "read", "infra.machines.1")
	cant(t, db, "i:alice", "write", "infra.machines")
}

func TestUsersGroupGrant(t *testing.T) {
	db := setupDB(t)
	err := data.CreateIdentity(db, tom)
	assert.NilError(t, err)

	err = data.CreateGroup(db, tomsGroup)
	assert.NilError(t, err)

	err = data.BindGroupIdentities(db, tomsGroup, *tom)
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", db)
	c.Set("identity", tom)
	c.Set("user", tom)

	grant(t, db, tom, tomsGroup.PolyID(), models.InfraUserRole, "infra")

	authDB, err := RequireInfraRole(c, models.InfraUserRole)
	assert.Check(t, err)
	assert.Check(t, authDB != nil)

	authDB, err = RequireInfraRole(c, models.InfraAdminRole)
	assert.Check(t, is.ErrorContains(err, ""))
	assert.Check(t, is.Nil(authDB))

	authDB, err = RequireInfraRole(c, models.InfraAdminRole, models.InfraUserRole)
	assert.Check(t, err)
	assert.Check(t, authDB != nil)
}

func grant(t *testing.T, db *gorm.DB, currentUser *models.Identity, subject uid.PolymorphicID, privilege, resource string) {
	err := data.CreateGrant(db, &models.Grant{
		Subject:   subject,
		Privilege: privilege,
		Resource:  resource,
		CreatedBy: currentUser.ID,
	})
	assert.NilError(t, err)
}

func can(t *testing.T, db *gorm.DB, subject uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, subject, privilege, resource)
	assert.NilError(t, err)
	assert.Assert(t, canAccess)
}

func cant(t *testing.T, db *gorm.DB, subject uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, subject, privilege, resource)
	assert.NilError(t, err)
	assert.Assert(t, !canAccess)
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
			"verify": func(t *testing.T, user *models.Identity, sessToken string, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "newusernewgroups@example.com", user.Name)
				assert.Assert(t, len(sessToken) != 0)
			},
		},
		"NewUserExistingGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				existingGroup1 := &models.Group{Name: "existing1"}
				existingGroup2 := &models.Group{Name: "existing2"}

				err := data.CreateGroup(db, existingGroup1)
				assert.NilError(t, err)

				err = data.CreateGroup(db, existingGroup2)
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "newuserexistinggroups@example.com",
					UserGroupsResp: []string{"existing1", "existing2"},
				}
			},
			"verify": func(t *testing.T, user *models.Identity, sessToken string, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "newuserexistinggroups@example.com", user.Name)
				assert.Assert(t, len(sessToken) != 0)

				assert.Assert(t, is.Len(user.Groups, 2))

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, is.Contains(groupNames, "existing1"))
				assert.Assert(t, is.Contains(groupNames, "existing2"))
			},
		},
		"ExistingUserNewGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				err := data.CreateIdentity(db, &models.Identity{Name: "existingusernewgroups@example.com", Kind: models.UserKind})
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "existingusernewgroups@example.com",
					UserGroupsResp: []string{"existingusernewgroups1", "existingusernewgroups2"},
				}
			},
			"verify": func(t *testing.T, user *models.Identity, sessToken string, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "existingusernewgroups@example.com", user.Name)
				assert.Assert(t, len(sessToken) != 0)

				assert.Assert(t, is.Len(user.Groups, 2))

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, is.Contains(groupNames, "existingusernewgroups1"))
				assert.Assert(t, is.Contains(groupNames, "existingusernewgroups2"))
			},
		},
		"ExistingUserExistingGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				err := data.CreateIdentity(db, &models.Identity{Name: "existinguserexistinggroups@example.com", Kind: models.UserKind})
				assert.NilError(t, err)

				err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups1"})
				assert.NilError(t, err)

				err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups2"})
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "existinguserexistinggroups@example.com",
					UserGroupsResp: []string{"existinguserexistinggroups1", "existinguserexistinggroups2"},
				}
			},
			"verify": func(t *testing.T, user *models.Identity, sessToken string, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "existinguserexistinggroups@example.com", user.Name)
				assert.Assert(t, len(sessToken) != 0)

				assert.Assert(t, is.Len(user.Groups, 2))

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, is.Contains(groupNames, "existinguserexistinggroups1"))
				assert.Assert(t, is.Contains(groupNames, "existinguserexistinggroups2"))
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
		assert.NilError(t, err)

		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gorm.DB) authn.OIDC)
			assert.Assert(t, ok)
			mockOIDC := setupFunc(t, db)

			u, sess, err := ExchangeAuthCodeForAccessKey(c, "123somecode", provider, mockOIDC, time.Now().Add(time.Minute), "example.com")

			verifyFunc, ok := v["verify"].(func(*testing.T, *models.Identity, string, error))
			assert.Assert(t, ok)

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
	assert.NilError(t, err)

	models.SymmetricKey = symmetricKey
}
