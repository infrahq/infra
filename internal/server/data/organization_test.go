package data

import (
	"reflect"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

var cmpOrgShallow = gocmp.Comparer(func(x, y models.Organization) bool {
	return x.Model.ID == y.Model.ID &&
		x.Name == y.Name &&
		x.Domain == y.Domain
})

var cmpProviderShallow = gocmp.Comparer(func(x, y models.Provider) bool {
	return x.Model.ID == y.Model.ID &&
		x.Name == y.Name &&
		reflect.DeepEqual(x.Scopes, y.Scopes) &&
		x.Kind == y.Kind &&
		x.OrganizationID == y.OrganizationID
})

func TestCreateOrganizationAndSetContext(t *testing.T) {
	pgsql := postgresDriver(t)
	db := setupDB(t, pgsql)

	org := &models.Organization{Name: "syndicate", Domain: "syndicate-123"}

	err := CreateOrganizationAndSetContext(db, org)
	assert.NilError(t, err)

	// db context is set
	ctxOrg := OrgFromContext(db.Statement.Context)
	assert.DeepEqual(t, org, ctxOrg, cmpOrgShallow)

	// org is created
	readOrg, err := GetOrganization(db, ByID(org.ID))
	assert.NilError(t, err)
	assert.DeepEqual(t, org, readOrg)

	// infra provider is created
	orgInfraIDP := InfraProvider(db)
	assert.NilError(t, err)

	expectedOrgInfraProviderIDP := &models.Provider{
		Model:              models.Model{ID: orgInfraIDP.Model.ID}, // does not matter
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Scopes:             models.CommaSeparatedStrings{},
		Name:               models.InternalInfraProviderName,
		Kind:               models.ProviderKindInfra,
	}
	assert.DeepEqual(t, orgInfraIDP, expectedOrgInfraProviderIDP, cmpProviderShallow)
}
