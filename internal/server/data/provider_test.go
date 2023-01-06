package data

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestCreateProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		providerDevelop := models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}

		err := CreateProvider(db, &providerDevelop)
		assert.NilError(t, err)

		actual, err := GetProvider(db, GetProviderOptions{ByID: providerDevelop.ID})
		assert.NilError(t, err)
		assert.DeepEqual(t, &providerDevelop, actual, cmpTimeWithDBPrecision, cmpopts.EquateEmpty())
	})
}

func createProviders(t *testing.T, db WriteTxn, providers ...*models.Provider) {
	for i := range providers {
		err := CreateProvider(db, providers[i])
		assert.NilError(t, err)
	}
}

func TestCreateProvider_DuplicateName(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		providerDevelop.ID = 0 // zero out the ID so that the conflict is on name
		err := CreateProvider(db, &providerDevelop)

		var uniqueConstraintErr UniqueConstraintError
		assert.Assert(t, errors.As(err, &uniqueConstraintErr), "error is wrong type %T", err)
		expected := UniqueConstraintError{Column: "name", Table: "providers"}
		assert.DeepEqual(t, uniqueConstraintErr, expected)
	})
}

func TestCreateProvider_RecreateWithDuplicateDomain(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		err := DeleteProviders(db, DeleteProvidersOptions{ByID: providerDevelop.ID})
		assert.NilError(t, err)

		err = CreateProvider(db, &models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta})
		assert.NilError(t, err)
	})
}

// TODO: combine CreateProvider tests into single func

func TestGetProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		providerDevelop := models.Provider{
			Name:             "okta-development",
			URL:              "example.com",
			ClientID:         "the-client-id",
			ClientSecret:     "the-client-secret",
			Kind:             models.ProviderKindOkta,
			AuthURL:          "https://example.com/auth",
			Scopes:           models.CommaSeparatedStrings{"scope1"},
			PrivateKey:       "private-key",
			ClientEmail:      "client@example.com",
			DomainAdminEmail: "admin@domain.example.com",
		}
		providerProduction := models.Provider{
			Name: "okta-production",
			URL:  "prod.okta.com",
			Kind: models.ProviderKindOkta,
		}
		providerDeleted := models.Provider{
			Name: "deleted",
			URL:  "somewhere.example.com",
			Kind: models.ProviderKindAzure,
		}
		providerDeleted.DeletedAt.Time = time.Now()
		providerDeleted.DeletedAt.Valid = true
		createProviders(t, db, &providerDevelop, &providerProduction, &providerDeleted)

		t.Run("default options", func(t *testing.T) {
			_, err := GetProvider(db, GetProviderOptions{})
			assert.ErrorContains(t, err, "an ID is required")
		})
		t.Run("by name", func(t *testing.T) {
			provider, err := GetProvider(db, GetProviderOptions{ByName: "okta-development"})
			assert.NilError(t, err)
			assert.Assert(t, provider.ID != 0)

			expected := models.Provider{
				Model: models.Model{
					ID:        provider.ID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
				Name:               "okta-development",
				URL:                "example.com",
				ClientID:           "the-client-id",
				ClientSecret:       "the-client-secret",
				Kind:               models.ProviderKindOkta,
				AuthURL:            "https://example.com/auth",
				Scopes:             models.CommaSeparatedStrings{"scope1"},
				PrivateKey:         "private-key",
				ClientEmail:        "client@example.com",
				DomainAdminEmail:   "admin@domain.example.com",
			}
			assert.DeepEqual(t, providerDevelop, expected, cmpModel)
		})
		t.Run("by id", func(t *testing.T) {
			provider, err := GetProvider(db, GetProviderOptions{ByName: "okta-development"})
			assert.NilError(t, err)
			assert.Assert(t, provider.ID != 0)
			assert.Equal(t, providerDevelop.URL, provider.URL)
		})
		t.Run("get deleted", func(t *testing.T) {
			_, err := GetProvider(db, GetProviderOptions{ByID: providerDeleted.ID})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("not found", func(t *testing.T) {
			_, err := GetProvider(db, GetProviderOptions{ByName: "does-not-exist"})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("from other organization", func(t *testing.T) {
			_, err := GetProvider(db, GetProviderOptions{
				ByName:           "okta-development",
				FromOrganization: 71234,
			})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestListProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		providerDev := &models.Provider{
			Name:      "okta-development",
			URL:       "example.com",
			Kind:      models.ProviderKindOkta,
			CreatedBy: 777,
		}
		providerProd := &models.Provider{
			Name:      "okta-production",
			URL:       "prod.okta.com",
			Kind:      models.ProviderKindOkta,
			CreatedBy: 777,
		}
		deleted := &models.Provider{
			Name:      "deleted",
			URL:       "somewhere.example.com",
			Kind:      models.ProviderKindOIDC,
			CreatedBy: 777,
		}
		deleted.DeletedAt.Valid = true
		deleted.DeletedAt.Time = time.Now()

		createProviders(t, db, providerDev, providerProd, deleted)

		otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		otherOrgProvider := &models.Provider{
			Name:               "okta-production",
			URL:                "prod.okta.com",
			Kind:               models.ProviderKindOkta,
			OrganizationMember: models.OrganizationMember{OrganizationID: otherOrg.ID},
		}
		createProviders(t, db, otherOrgProvider)

		providerInfra := InfraProvider(db)

		t.Run("default", func(t *testing.T) {
			actual, err := ListProviders(db, ListProvidersOptions{})
			assert.NilError(t, err)

			expected := []models.Provider{*providerInfra, *providerDev, *providerProd}
			assert.DeepEqual(t, expected, actual, cmpModelByID)
		})
		t.Run("exclude infra provider", func(t *testing.T) {
			actual, err := ListProviders(db, ListProvidersOptions{
				ExcludeInfraProvider: true,
			})
			assert.NilError(t, err)

			expected := []models.Provider{*providerDev, *providerProd}
			assert.DeepEqual(t, expected, actual, cmpModelByID)
		})
		t.Run("by name", func(t *testing.T) {
			actual, err := ListProviders(db, ListProvidersOptions{
				ByName: "okta-development",
			})
			assert.NilError(t, err)

			expected := []models.Provider{*providerDev}
			assert.DeepEqual(t, expected, actual, cmpModelByID)
		})
		t.Run("by IDs", func(t *testing.T) {
			actual, err := ListProviders(db, ListProvidersOptions{
				ByIDs: []uid.ID{providerDev.ID, providerInfra.ID},
			})
			assert.NilError(t, err)

			expected := []models.Provider{*providerInfra, *providerDev}
			assert.DeepEqual(t, expected, actual, cmpModelByID)
		})
		t.Run("created by and notIDs", func(t *testing.T) {
			actual, err := ListProviders(db, ListProvidersOptions{
				CreatedBy: 777,
				NotIDs:    []uid.ID{providerDev.ID},
			})
			assert.NilError(t, err)

			expected := []models.Provider{*providerProd}
			assert.DeepEqual(t, expected, actual, cmpModelByID)
		})
		t.Run("pagination", func(t *testing.T) {
			page := Pagination{Page: 2, Limit: 2}
			actual, err := ListProviders(db, ListProvidersOptions{Pagination: &page})
			assert.NilError(t, err)

			expected := []models.Provider{*providerProd}
			assert.DeepEqual(t, expected, actual, cmpModelByID)
			assert.Equal(t, page.TotalCount, 3)
		})
		t.Run("pagination with filter", func(t *testing.T) {
			page := Pagination{Page: 1, Limit: 2}
			actual, err := ListProviders(db, ListProvidersOptions{
				Pagination: &page,
				ByIDs:      []uid.ID{providerInfra.ID, providerProd.ID, providerDev.ID},
			})
			assert.NilError(t, err)

			expected := []models.Provider{*providerInfra, *providerDev}
			assert.DeepEqual(t, expected, actual, cmpModelByID)
			assert.Equal(t, page.TotalCount, 3)
		})
	})
}

func TestUpdateProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			orig := models.Provider{
				Name: "idp",
				Kind: models.ProviderKindGoogle,
			}
			createProviders(t, tx, &orig)

			updated := models.Provider{
				Model:            models.Model{ID: orig.ID},
				Name:             "new-name",
				URL:              "https://example.com/idp",
				ClientID:         "client-id",
				ClientSecret:     "client-secret",
				CreatedBy:        777,
				Kind:             models.ProviderKindAzure,
				AuthURL:          "https://example.com/auth",
				Scopes:           []string{"one", "two"},
				PrivateKey:       "private-key",
				ClientEmail:      "client-email@example.com",
				DomainAdminEmail: "domain-admin-email@example.com",
			}
			err := UpdateProvider(tx, &updated)
			assert.NilError(t, err)

			actual, err := GetProvider(tx, GetProviderOptions{ByID: orig.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, &updated, cmpTimeWithDBPrecision)
		})
		t.Run("name conflict", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			orig := models.Provider{Name: "idp", Kind: models.ProviderKindGoogle}
			other := models.Provider{Name: "taken", Kind: models.ProviderKindOIDC}
			createProviders(t, tx, &orig, &other)

			err := UpdateProvider(tx, &models.Provider{
				Model: models.Model{ID: orig.ID},
				Name:  other.Name,
				Kind:  orig.Kind,
			})

			var ucErr UniqueConstraintError
			assert.Assert(t, errors.As(err, &ucErr))
			expected := UniqueConstraintError{Column: "name", Table: "providers"}
			assert.DeepEqual(t, ucErr, expected)
		})
	})
}

func TestDeleteProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{}
			providerProduction = models.Provider{}
			pu                 = &models.ProviderUser{}
			user               = &models.Identity{}
		)

		otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		setup := func(t *testing.T) *Transaction {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			providerDevelop = models.Provider{
				Name: "okta-development",
				URL:  "example.com",
				Kind: models.ProviderKindOkta,
			}
			providerProduction = models.Provider{
				Name: "okta-production",
				URL:  "prod.okta.com",
				Kind: models.ProviderKindOkta,
			}
			createProviders(t, tx, &providerDevelop, &providerProduction)

			user = &models.Identity{Name: "joe@example.com"}
			err := CreateIdentity(tx, user)
			assert.NilError(t, err)

			pu, err = CreateProviderUser(tx, &providerDevelop, user)
			assert.NilError(t, err)
			return tx
		}

		t.Run("Deletes work", func(t *testing.T) {
			tx := setup(t)
			err := DeleteProviders(tx, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetProvider(tx, GetProviderOptions{ByName: providerDevelop.Name})
			assert.Error(t, err, "record not found")

			t.Run("provider users are removed", func(t *testing.T) {
				_, err = GetProviderUser(tx, pu.ProviderID, pu.IdentityID)
				assert.Error(t, err, "record not found")
			})

			t.Run("user is removed when last providerUser is removed", func(t *testing.T) {
				_, err = GetIdentity(tx, GetIdentityOptions{ByID: pu.IdentityID})
				assert.Error(t, err, "record not found")
			})
		})

		t.Run("access keys issued using deleted provider are revoked", func(t *testing.T) {
			tx := setup(t)

			key := &models.AccessKey{
				Name:       "test key",
				IssuedFor:  user.ID,
				ProviderID: providerDevelop.ID,
				ExpiresAt:  time.Now().Add(5 * time.Minute),
			}

			_, err := CreateAccessKey(tx, key)
			assert.NilError(t, err)

			err = DeleteProviders(tx, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetAccessKeyByKeyID(tx, key.KeyID)
			assert.ErrorContains(t, err, "record not found")
		})

		t.Run("access keys issued using different provider from deleted are NOT revoked", func(t *testing.T) {
			tx := setup(t)

			_, err := CreateProviderUser(tx, &providerProduction, user)
			assert.NilError(t, err)

			key := &models.AccessKey{
				Name:       "test key",
				IssuedFor:  user.ID,
				ProviderID: providerProduction.ID,
				ExpiresAt:  time.Now().Add(5 * time.Minute),
			}

			_, err = CreateAccessKey(tx, key)
			assert.NilError(t, err)

			err = DeleteProviders(tx, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetAccessKeyByKeyID(tx, key.KeyID)
			assert.NilError(t, err)

			// clean up
			err = DeleteProviders(tx, DeleteProvidersOptions{ByID: providerProduction.ID})
			assert.NilError(t, err)
		})

		t.Run("user is not removed if there are other providerUsers", func(t *testing.T) {
			tx := setup(t)

			pu, err := CreateProviderUser(tx, &providerProduction, user)
			assert.NilError(t, err)

			err = DeleteProviders(tx, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetIdentity(tx, GetIdentityOptions{ByID: pu.IdentityID})
			assert.NilError(t, err)
		})

		t.Run("user is not removed if there are social providerUsers", func(t *testing.T) {
			tx := setup(t)

			pu, err := CreateProviderUser(tx, &models.Provider{Model: models.Model{ID: models.InternalGoogleProviderID}}, user)
			assert.NilError(t, err)

			err = DeleteProviders(tx, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetIdentity(tx, GetIdentityOptions{ByID: pu.IdentityID})
			assert.NilError(t, err)
		})

		t.Run("delete in wrong org", func(t *testing.T) {
			tx := setup(t)

			otherOrgTx := tx.WithOrgID(otherOrg.ID)
			err := DeleteProviders(otherOrgTx, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetIdentity(tx, GetIdentityOptions{ByID: user.ID})
			assert.NilError(t, err)

			_, err = GetProviderUser(tx, providerDevelop.ID, user.ID)
			assert.NilError(t, err)
		})
	})
}

func TestCountProvidersByKind(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createProviders(t, db,
			&models.Provider{Name: "oidc", Kind: "oidc"},
			&models.Provider{Name: "okta", Kind: "okta"},
			&models.Provider{Name: "okta2", Kind: "okta"},
			&models.Provider{Name: "azure", Kind: "azure"},
			&models.Provider{Name: "google", Kind: "google"},
		)

		actual, err := CountProvidersByKind(db)
		assert.NilError(t, err)

		expected := []providersCount{
			{Kind: "azure", Count: 1},
			{Kind: "google", Count: 1},
			{Kind: "oidc", Count: 1},
			{Kind: "okta", Count: 2},
		}

		assert.DeepEqual(t, actual, expected)
	})
}

func TestCountAllProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createProviders(t, db,
			&models.Provider{Name: "oidc", Kind: "oidc"},
			&models.Provider{Name: "azure", Kind: "azure"},
			&models.Provider{Name: "google", Kind: "google"},
		)

		actual, err := CountAllProviders(db)
		assert.NilError(t, err)
		assert.Equal(t, actual, int64(4)) // 3 + infra provider
	})
}
