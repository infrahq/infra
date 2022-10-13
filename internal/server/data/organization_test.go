package data

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "syndicate", Domain: "syndicate-123"}

		err := CreateOrganization(db, org)
		assert.NilError(t, err)

		tx := &Transaction{DB: db.DB, MetadataSource: metadata{orgID: org.ID}}

		// org is created
		readOrg, err := GetOrganization(db, ByID(org.ID))
		assert.NilError(t, err)
		assert.DeepEqual(t, org, readOrg, cmpTimeWithDBPrecision)

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
