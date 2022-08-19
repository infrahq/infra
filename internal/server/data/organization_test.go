package data

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

// PostgreSQL only has microsecond precision
var cmpTimeWithDBPrecision = cmpopts.EquateApproxTime(time.Microsecond)

func TestCreateOrganizationAndSetContext(t *testing.T) {
	pgsql := postgresDriver(t)
	db := setupDB(t, pgsql)

	org := &models.Organization{Name: "syndicate", Domain: "syndicate-123"}

	err := CreateOrganizationAndSetContext(db, org)
	assert.NilError(t, err)

	// db context is set
	ctxOrg := OrgFromContext(db.Statement.Context)
	assert.DeepEqual(t, org, ctxOrg, cmpTimeWithDBPrecision)

	// org is created
	readOrg, err := GetOrganization(db, ByID(org.ID))
	assert.NilError(t, err)
	assert.DeepEqual(t, org, readOrg, cmpTimeWithDBPrecision)

	// infra provider is created
	orgInfraIDP := InfraProvider(db)
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
	connector, err := GetIdentity(db, ByName(models.InternalInfraConnectorIdentityName))
	assert.NilError(t, err)

	expectedConnector := &models.Identity{
		Model:              connector.Model,
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               models.InternalInfraConnectorIdentityName,
		CreatedBy:          models.CreatedBySystem,
	}
	assert.DeepEqual(t, connector, expectedConnector)

	connectorGrant, err := GetGrant(db, BySubject(connector.PolyID()), ByPrivilege(models.InfraAdminRole), ByResource("infra"))
	assert.NilError(t, err)

	expectedConnectorGrant := &models.Grant{
		Model:              connectorGrant.Model,
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Subject:            connector.PolyID(),
		Privilege:          models.InfraAdminRole,
		Resource:           "infra",
		CreatedBy:          models.CreatedBySystem,
	}
	assert.DeepEqual(t, connectorGrant, expectedConnectorGrant)
}
