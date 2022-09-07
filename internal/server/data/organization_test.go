package data

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateOrganization(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "syndicate", Domain: "syndicate-123"}

		err := CreateOrganization(db, org)
		assert.NilError(t, err)

		tx := &Transaction{DB: db.DB, orgID: org.ID}

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
		connector, err := GetIdentity(tx, ByName(models.InternalInfraConnectorIdentityName))
		assert.NilError(t, err)

		expectedConnector := &models.Identity{
			Model:              connector.Model,
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			Name:               models.InternalInfraConnectorIdentityName,
			CreatedBy:          models.CreatedBySystem,
		}
		assert.DeepEqual(t, connector, expectedConnector)

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
		}
		assert.DeepEqual(t, connectorGrant, expectedConnectorGrant)
	})
}
