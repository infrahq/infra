package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/infrahq/infra/internal/server/models"
)

// PostgreSQL only has microsecond precision
var cmpTimeWithDBPrecision = cmpopts.EquateApproxTime(time.Millisecond)

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
	assert.DeepEqual(t, org, readOrg)

	// infra provider is created
	orgInfraIDP := InfraProvider(db)
	assert.NilError(t, err)

	expectedOrgInfraProviderIDP := &models.Provider{
		Model:              orgInfraIDP.Model, // does not matter
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Scopes:             models.CommaSeparatedStrings{},
		Name:               models.InternalInfraProviderName,
		Kind:               models.ProviderKindInfra,
	}
	assert.DeepEqual(t, orgInfraIDP, expectedOrgInfraProviderIDP, cmpTimeWithDBPrecision)
}
