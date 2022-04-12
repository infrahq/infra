package access

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ssoroka/slice"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal"
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
		Resource:  ResourceInfraAPI,
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

	authDB, err := RequireInfraRole(c, models.InfraAdminRole)
	assert.ErrorIs(t, err, internal.ErrForbidden)
	assert.Assert(t, authDB == nil)

	grant(t, db, tom, tomsGroup.PolyID(), models.InfraAdminRole, "infra")

	authDB, err = RequireInfraRole(c, models.InfraAdminRole)
	assert.NilError(t, err)
	assert.Assert(t, authDB != nil)
}

func TestInfraRequireInfraRole(t *testing.T) {
	db := setupDB(t)

	setup := func(t *testing.T, infraRole string) *gin.Context {
		testIdentity := &models.Identity{Name: fmt.Sprintf("infra-%s-%s", infraRole, time.Now()), Kind: models.MachineKind}

		err := data.CreateIdentity(db, testIdentity)
		assert.NilError(t, err)

		err = data.CreateGrant(db, &models.Grant{Subject: testIdentity.PolyID(), Privilege: infraRole, Resource: ResourceInfraAPI})
		assert.NilError(t, err)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("db", db)
		c.Set("identity", testIdentity)

		return c
	}

	t.Run("has specific required role", func(t *testing.T) {
		c := setup(t, models.InfraAdminRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole)
		assert.NilError(t, err)
		assert.Assert(t, authDB != nil)
	})

	t.Run("does not have specific required role", func(t *testing.T) {
		c := setup(t, models.InfraViewRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole)
		assert.Error(t, err, "forbidden: requestor does not have required grant")
		assert.Assert(t, authDB == nil)
	})

	t.Run("has required role in list", func(t *testing.T) {
		c := setup(t, models.InfraViewRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole)
		assert.NilError(t, err)
		assert.Assert(t, authDB != nil)
	})

	t.Run("does not have required role in list", func(t *testing.T) {
		c := setup(t, models.InfraViewRole)

		authDB, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraConnectorRole)
		assert.Error(t, err, "forbidden: requestor does not have required grant")
		assert.Assert(t, authDB == nil)
	})
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

func (o *mockOIDCImplementation) RefreshAccessToken(providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	// never update
	return string(providerUser.AccessToken), &providerUser.ExpiresAt, nil
}

func (m *mockOIDCImplementation) GetUserInfo(providerUser *models.ProviderUser) (*authn.UserInfo, error) {
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
		"ExistingUserGroupsWithNewGroups": {
			"setup": func(t *testing.T, db *gorm.DB) authn.OIDC {
				user := &models.Identity{Name: "eugwnw@example.com"}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)
				err = db.Model(user).Association("Groups").Append([]models.Group{{Name: "Foo"}, {Name: "existing3"}})
				assert.NilError(t, err)
				assert.Assert(t, len(user.Groups) == 2)

				err = data.SaveIdentity(db, user)
				assert.NilError(t, err)
				g, err := data.GetGroup(db, data.ByName("Foo"))
				assert.NilError(t, err)
				assert.Assert(t, g != nil)

				user, err = data.GetIdentity(db.Preload("Groups"), data.ByID(user.ID))
				assert.NilError(t, err)
				assert.Assert(t, user != nil)
				assert.Assert(t, len(user.Groups) == 2)

				p, err := data.GetProvider(db, data.ByName("mockoidc"))
				assert.NilError(t, err)

				pu, err := data.CreateProviderUser(db, p, user)
				assert.NilError(t, err)

				pu.Groups = []string{"existing3"}
				err = db.Save(pu).Error
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "eugwnw@example.com",
					UserGroupsResp: []string{"existinguserexistinggroups1", "existinguserexistinggroups2"},
				}
			},
			"verify": func(t *testing.T, user *models.Identity, sessToken string, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "eugwnw@example.com", user.Name)
				assert.Assert(t, sessToken != "")

				assert.Assert(t, len(user.Groups) == 3)

				var groupNames []string
				for _, g := range user.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, slice.Contains(groupNames, "Foo"))
				assert.Assert(t, slice.Contains(groupNames, "existinguserexistinggroups1"))
				assert.Assert(t, slice.Contains(groupNames, "existinguserexistinggroups2"))
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

			if err == nil {
				// make sure the associations are still set when you reload the object.
				u, err = data.GetIdentity(db.Preload("Groups"), data.ByID(u.ID))
				assert.NilError(t, err)

				verifyFunc(t, u, sess, err)
			}
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
