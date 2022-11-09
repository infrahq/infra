package data

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{
			Name:           "syndicate",
			Domain:         "syndicate-123",
			CreatedBy:      777,
			AllowedDomains: models.CommaSeparatedStrings{"example.com", "infrahq.com"},
		}

		err := CreateOrganization(db, org)
		assert.NilError(t, err)

		tx := txnForTestCase(t, db, org.ID)

		// org is created
		actual, err := GetOrganization(db, GetOrganizationOptions{ByID: org.ID})
		assert.NilError(t, err)
		expected := &models.Organization{
			Model: models.Model{
				ID:        12345,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name:           "syndicate",
			Domain:         "syndicate-123",
			CreatedBy:      777,
			AllowedDomains: models.CommaSeparatedStrings{"example.com", "infrahq.com"},
		}
		assert.DeepEqual(t, expected, actual, cmpModel)

		// infra provider is created
		orgInfraIDP := InfraProvider(tx)
		assert.NilError(t, err)

		expectedOrgInfraProviderIDP := &models.Provider{
			Model:              orgInfraIDP.Model, // does not matter
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			Scopes:             models.CommaSeparatedStrings{},
			Name:               models.InternalInfraProviderName,
			Kind:               models.ProviderKindInfra,
			CreatedBy:          models.CreatedBySystem,
		}
		assert.DeepEqual(t, orgInfraIDP, expectedOrgInfraProviderIDP, cmpTimeWithDBPrecision)

		// the org connector is created and granted approprite access
		connector, err := GetIdentity(tx, GetIdentityOptions{ByName: models.InternalInfraConnectorIdentityName})
		assert.NilError(t, err)

		expectedConnector := &models.Identity{
			Model:              connector.Model,
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			Name:               models.InternalInfraConnectorIdentityName,
			CreatedBy:          models.CreatedBySystem,
			VerificationToken:  "abcde12345",
			SSHUsername:        "connector",
		}
		assert.DeepEqual(t, connector, expectedConnector, anyValidToken)

		connectorGrant, err := GetGrant(tx, GetGrantOptions{
			BySubject:   connector.PolyID(),
			ByPrivilege: models.InfraConnectorRole,
			ByResource:  "infra",
		})
		assert.NilError(t, err)
		expectedConnectorGrant := &models.Grant{
			Model:              connectorGrant.Model,
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			Subject:            connector.PolyID(),
			Privilege:          models.InfraConnectorRole,
			Resource:           "infra",
			CreatedBy:          models.CreatedBySystem,
			UpdateIndex:        10001,
		}
		assert.DeepEqual(t, connectorGrant, expectedConnectorGrant)
	})
}

var anyValidToken = cmp.Comparer(func(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) > 0 && b == "abcde12345" {
		return true
	}
	if len(b) > 0 && a == "abcde12345" {
		return true
	}
	return false
})

func TestGetOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		first := &models.Organization{
			Name:           "first",
			Domain:         "first.example.com",
			AllowedDomains: []string{},
		}
		deleted := &models.Organization{
			Name:           "deleted",
			Domain:         "none.example.com",
			AllowedDomains: []string{},
		}
		deleted.DeletedAt.Valid = true
		deleted.DeletedAt.Time = time.Now()
		assert.NilError(t, CreateOrganization(tx, first))
		assert.NilError(t, CreateOrganization(tx, deleted))

		t.Run("default options", func(t *testing.T) {
			_, err := GetOrganization(tx, GetOrganizationOptions{})
			assert.ErrorContains(t, err, "an ID or domain is required")
		})
		t.Run("by id", func(t *testing.T) {
			actual, err := GetOrganization(tx, GetOrganizationOptions{ByID: first.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, first, cmpTimeWithDBPrecision)
		})
		t.Run("by domain", func(t *testing.T) {
			actual, err := GetOrganization(tx, GetOrganizationOptions{
				ByDomain: "first.example.com",
			})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, first, cmpTimeWithDBPrecision)
		})
		t.Run("deleted org", func(t *testing.T) {
			_, err := GetOrganization(tx, GetOrganizationOptions{ByID: deleted.ID})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("does not exist", func(t *testing.T) {
			_, err := GetOrganization(tx, GetOrganizationOptions{ByID: 171717})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestUpdateOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, 0)

		past := time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC)
		org := &models.Organization{
			Model: models.Model{
				CreatedAt: past,
				UpdatedAt: past,
			},
			Name:           "second",
			Domain:         "second.example.com",
			AllowedDomains: []string{},
		}
		err := CreateOrganization(db, org)
		assert.NilError(t, err)

		updated := *org // shallow copy
		updated.Domain = "third.example.com"
		updated.Name = "next"
		updated.CreatedBy = 7123
		updated.AllowedDomains = []string{"example.com"}

		err = UpdateOrganization(tx, &updated)
		assert.NilError(t, err)

		actual, err := GetOrganization(tx, GetOrganizationOptions{ByID: org.ID})
		assert.NilError(t, err)

		expected := &models.Organization{
			Model: models.Model{
				ID:        org.ID,
				CreatedAt: past,
				UpdatedAt: time.Now(),
			},
			Name:           "next",
			Domain:         "third.example.com",
			CreatedBy:      7123,
			AllowedDomains: []string{"example.com"},
		}
		assert.DeepEqual(t, expected, actual, cmpModel)
	})
}

func TestListOrganizations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		first := &models.Organization{
			Name:           "first",
			Domain:         "first.example.com",
			AllowedDomains: []string{},
		}
		second := &models.Organization{
			Name:           "second",
			Domain:         "second.example.com",
			AllowedDomains: []string{},
		}
		deleted := &models.Organization{
			Name:           "deleted",
			Domain:         "none.example.com",
			AllowedDomains: []string{},
		}
		deleted.DeletedAt.Valid = true
		deleted.DeletedAt.Time = time.Now()
		assert.NilError(t, CreateOrganization(tx, first))
		assert.NilError(t, CreateOrganization(tx, second))
		assert.NilError(t, CreateOrganization(tx, deleted))

		t.Run("defaults", func(t *testing.T) {
			actual, err := ListOrganizations(tx, ListOrganizationsOptions{})
			assert.NilError(t, err)

			expected := []models.Organization{*db.DefaultOrg, *first, *second}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("by name", func(t *testing.T) {
			actual, err := ListOrganizations(tx, ListOrganizationsOptions{
				ByName: "second",
			})
			assert.NilError(t, err)

			expected := []models.Organization{*second}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("with pagination", func(t *testing.T) {
			page := Pagination{Limit: 2, Page: 2}
			actual, err := ListOrganizations(tx, ListOrganizationsOptions{
				Pagination: &page,
			})
			assert.NilError(t, err)

			expected := []models.Organization{*second}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
			assert.Equal(t, page.TotalCount, 3)
		})
	})
}

func TestDeleteOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{
			Name:   "first",
			Domain: "first.example.com",
		}

		err := CreateOrganization(db, org)
		assert.NilError(t, err)

		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, 0)
			err := DeleteOrganization(tx, org.ID)
			assert.NilError(t, err)

			_, err = GetOrganization(tx, GetOrganizationOptions{ByID: org.ID})
			assert.ErrorIs(t, err, internal.ErrNotFound)

			// delete again to check idempotence
			err = DeleteOrganization(tx, org.ID)
			assert.NilError(t, err)
		})
	})
}

func TestCountOrganizations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		assert.NilError(t, CreateOrganization(db, &models.Organization{
			Name:   "first",
			Domain: "first.example.com",
		}))
		assert.NilError(t, CreateOrganization(db, &models.Organization{
			Name:   "second",
			Domain: "second.example.com",
		}))

		actual, err := CountOrganizations(db)
		assert.NilError(t, err)
		assert.Equal(t, actual, int64(3)) // 2 + default org
	})
}
