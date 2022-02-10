package data

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

var tom = &models.User{Email: "tom@infrahq.com"}

func TestBasicGrant(t *testing.T) {
	db := setup(t)
	err := CreateUser(db, tom)
	require.NoError(t, err)

	grant(t, db, tom, "u:steven", "read", "infra.groups.1")

	can(t, db, "u:steven", "read", "infra.groups.1")
	cant(t, db, "u:steven", "read", "infra.groups.2")
	cant(t, db, "u:steven", "write", "infra.groups.1")

	grant(t, db, tom, "u:steven", "read", "infra.groups.*")
	can(t, db, "u:steven", "read", "infra.groups.1")
	can(t, db, "u:steven", "read", "infra.groups.2")
	cant(t, db, "u:steven", "write", "infra.groups.1")
}

func grant(t *testing.T, db *gorm.DB, currentUser *models.User, identity uid.PolymorphicID, privilege, resource string) {
	err := CreateGrant(db, &models.Grant{
		Identity:  identity,
		Privilege: privilege,
		Resource:  resource,
		CreatedBy: currentUser.ID,
	})
	require.NoError(t, err)
}

func can(t *testing.T, db *gorm.DB, identity uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, identity, privilege, resource)
	require.NoError(t, err)
	require.True(t, canAccess)
}

func cant(t *testing.T, db *gorm.DB, identity uid.PolymorphicID, privilege, resource string) {
	canAccess, err := Can(db, identity, privilege, resource)
	require.NoError(t, err)
	require.False(t, canAccess)
}

func TestWildcardCombinations(t *testing.T) {
	tests := []struct {
		input  string
		output []string
	}{
		{"infra.foo.1", []string{"infra.foo.1", "infra.foo.*", "infra.*"}},
		{"k8s.mycluster.mynamespace.apiPath.secrets.1", []string{"k8s.mycluster.mynamespace.apiPath.secrets.1", "k8s.mycluster.mynamespace.apiPath.secrets.*", "k8s.mycluster.mynamespace.apiPath.*", "k8s.mycluster.mynamespace.*.secrets.1", "k8s.mycluster.mynamespace.*.secrets.*", "k8s.mycluster.mynamespace.*", "k8s.mycluster.*.apiPath.secrets.1", "k8s.mycluster.*.apiPath.secrets.*", "k8s.mycluster.*.apiPath.*", "k8s.mycluster.*.*.secrets.1", "k8s.mycluster.*.*.secrets.*", "k8s.mycluster.*", "k8s.*.mynamespace.apiPath.secrets.1", "k8s.*.mynamespace.apiPath.secrets.*", "k8s.*.mynamespace.apiPath.*", "k8s.*.mynamespace.*.secrets.1", "k8s.*.mynamespace.*.secrets.*", "k8s.*.mynamespace.*", "k8s.*.*.apiPath.secrets.1", "k8s.*.*.apiPath.secrets.*", "k8s.*.*.apiPath.*", "k8s.*.*.*.secrets.1", "k8s.*.*.*.secrets.*", "k8s.*"}},
	}
	for _, test := range tests {
		require.Equal(t, test.output, wildcardCombinations(test.input))
	}
}
