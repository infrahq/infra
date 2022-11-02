package data

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{
			Name:      "syndicate",
			Domain:    "syndicate-123",
			CreatedBy: 777,
		}

		err := CreateOrganization(db, org)
		assert.NilError(t, err)

		tx := txnForTestCase(t, db, org.ID)

		// org is created
		actual, err := GetOrganization(db, ByID(org.ID))
		assert.NilError(t, err)
		expected := &models.Organization{
			Model: models.Model{
				ID:        12345,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name:      "syndicate",
			Domain:    "syndicate-123",
			CreatedBy: 777,
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

func TestUpdateOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, 0)

		past := time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC)
		org := &models.Organization{
			Model: models.Model{
				CreatedAt: past,
				UpdatedAt: past,
			},
			Name:   "second",
			Domain: "second.example.com",
		}
		err := CreateOrganization(db, org)
		assert.NilError(t, err)

		updated := *org // shallow copy
		updated.Domain = "third.example.com"
		updated.Name = "next"
		updated.CreatedBy = 7123

		err = UpdateOrganization(tx, &updated)
		assert.NilError(t, err)

		actual, err := GetOrganization(tx, ByID(org.ID))
		assert.NilError(t, err)

		expected := &models.Organization{
			Model: models.Model{
				ID:        org.ID,
				CreatedAt: past,
				UpdatedAt: time.Now(),
			},
			Name:      "next",
			Domain:    "third.example.com",
			CreatedBy: 7123,
		}
		assert.DeepEqual(t, expected, actual, cmpModel)
	})
}
