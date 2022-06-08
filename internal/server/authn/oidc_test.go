package authn

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ssoroka/slice"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// mockOIDC is a mock oidc identity provider
type mockOIDCImplementation struct {
	UserEmailResp  string
	UserGroupsResp []string
}

func (m *mockOIDCImplementation) Validate() error {
	return nil
}

func (m *mockOIDCImplementation) ExchangeAuthCodeForProviderTokens(code string) (acc, ref string, exp time.Time, email string, err error) {
	return "acc", "ref", exp, m.UserEmailResp, nil
}

func (o *mockOIDCImplementation) RefreshAccessToken(providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	// never update
	return string(providerUser.AccessToken), &providerUser.ExpiresAt, nil
}

func (m *mockOIDCImplementation) GetUserInfo(providerUser *models.ProviderUser) (*InfoClaims, error) {
	return &InfoClaims{Email: m.UserEmailResp, Groups: m.UserGroupsResp}, nil
}

func TestOIDCAuthenticate(t *testing.T) {
	// setup
	db := setupDB(t)

	mocktaProvider := models.Provider{Name: "mockta"}
	err := data.CreateProvider(db, &mocktaProvider)
	assert.NilError(t, err)

	oidc := &mockOIDCImplementation{
		UserEmailResp:  "bruce@example.com",
		UserGroupsResp: []string{"Everyone", "developers"},
	}

	t.Run("invalid provider", func(t *testing.T) {
		unknownProviderOIDCAuthn := NewOIDCAuthentication(uid.New(), "localhost:8031", "1234", oidc)
		_, _, err := unknownProviderOIDCAuthn.Authenticate(db)

		assert.ErrorIs(t, err, internal.ErrNotFound)
	})

	t.Run("successful authentication", func(t *testing.T) {
		oidcAuthn := NewOIDCAuthentication(mocktaProvider.ID, "localhost:8031", "1234", oidc)
		identity, provider, err := oidcAuthn.Authenticate(db)

		assert.NilError(t, err)
		// user should be created
		assert.Equal(t, identity.Name, "bruce@example.com")

		groups := make(map[string]bool)
		for _, g := range identity.Groups {
			groups[g.Name] = true
		}
		assert.Assert(t, len(identity.Groups) == 2)
		assert.Equal(t, groups["Everyone"], true)
		assert.Equal(t, groups["developers"], true)

		assert.Equal(t, provider.ID, mocktaProvider.ID)
	})
}

func TestValidateInvalidURL(t *testing.T) {
	oidc := NewOIDC("invalid.example.com", "some_client_id", "some_client_secret", "http://localhost:8301")

	err := oidc.Validate()
	assert.ErrorIs(t, err, ErrInvalidProviderURL)
}

func TestExchangeAuthCodeForProviderTokens(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"NewUserNewGroups": {
			"setup": func(t *testing.T, db *gorm.DB) OIDC {
				return &mockOIDCImplementation{
					UserEmailResp:  "newusernewgroups@example.com",
					UserGroupsResp: []string{"Everyone", "developers"},
				}
			},
			"verify": func(t *testing.T, user *models.Identity, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "newusernewgroups@example.com", user.Name)
			},
		},
		"NewUserExistingGroups": {
			"setup": func(t *testing.T, db *gorm.DB) OIDC {
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
			"verify": func(t *testing.T, user *models.Identity, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "newuserexistinggroups@example.com", user.Name)

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
			"setup": func(t *testing.T, db *gorm.DB) OIDC {
				err := data.CreateIdentity(db, &models.Identity{Name: "existingusernewgroups@example.com"})
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "existingusernewgroups@example.com",
					UserGroupsResp: []string{"existingusernewgroups1", "existingusernewgroups2"},
				}
			},
			"verify": func(t *testing.T, user *models.Identity, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "existingusernewgroups@example.com", user.Name)

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
			"setup": func(t *testing.T, db *gorm.DB) OIDC {
				err := data.CreateIdentity(db, &models.Identity{Name: "existinguserexistinggroups@example.com"})
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
			"verify": func(t *testing.T, user *models.Identity, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "existinguserexistinggroups@example.com", user.Name)

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
			"setup": func(t *testing.T, db *gorm.DB) OIDC {
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
			"verify": func(t *testing.T, user *models.Identity, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "eugwnw@example.com", user.Name)

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
		db := setupDB(t)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("db", db)

		// setup fake identity provider
		provider := &models.Provider{Name: "mockoidc", URL: "mockOIDC.example.com"}
		err := data.CreateProvider(db, provider)
		assert.NilError(t, err)

		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gorm.DB) OIDC)
			assert.Assert(t, ok)
			mockOIDC := setupFunc(t, db)

			loginMethod := NewOIDCAuthentication(provider.ID, "mockOIDC.example.com/redirect", "AAA", mockOIDC)

			u, _, err := loginMethod.Authenticate(db)

			verifyFunc, ok := v["verify"].(func(*testing.T, *models.Identity, error))
			assert.Assert(t, ok)

			verifyFunc(t, u, err)

			if err == nil {
				// make sure the associations are still set when you reload the object.
				u, err = data.GetIdentity(db.Preload("Groups"), data.ByID(u.ID))
				assert.NilError(t, err)

				verifyFunc(t, u, err)
			}
		})
	}
}

func TestUserInfo(t *testing.T) {
	t.Run("no email and no name fails validation", func(t *testing.T) {
		claims := &InfoClaims{}
		err := claims.validate()
		assert.ErrorContains(t, err, "name or email are required")
	})
	t.Run("groups are not required", func(t *testing.T) {
		claims := &InfoClaims{Email: "hello@example.com"}
		err := claims.validate()
		assert.NilError(t, err)
	})
}
