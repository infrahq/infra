package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
)

var (
	admin = models.Grant{Kind: "kubernetes", Kubernetes: models.GrantKubernetes{Kind: "role", Name: "admin"}}
	view  = models.Grant{Kind: "kubernetes", Kubernetes: models.GrantKubernetes{Kind: "role", Name: "view"}}
	edit  = models.Grant{Kind: "kubernetes", Kubernetes: models.GrantKubernetes{Kind: "role", Name: "edit"}}
)

func TestGrant(t *testing.T) {
	db := setup(t)

	err := db.Create(&admin).Error
	require.NoError(t, err)

	var grant models.Grant
	err = db.Preload("Kubernetes").First(&grant, &models.Grant{Kind: "kubernetes"}).Error
	require.NoError(t, err)
	require.Equal(t, models.GrantKindKubernetes, grant.Kind)
	require.Equal(t, models.GrantKubernetesKindRole, grant.Kubernetes.Kind)
	require.Equal(t, "admin", grant.Kubernetes.Name)
}

func TestCreateGrant(t *testing.T) {
	db := setup(t)

	grant, err := CreateGrant(db, &admin)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, grant.ID)
	require.Equal(t, admin.Kind, grant.Kind)
	require.Equal(t, admin.Kubernetes.Kind, grant.Kubernetes.Kind)
	require.Equal(t, admin.Kubernetes.Name, grant.Kubernetes.Name)
}

func createGrants(t *testing.T, db *gorm.DB, grants ...models.Grant) {
	for i := range grants {
		_, err := CreateGrant(db, &grants[i])
		require.NoError(t, err)
	}
}

func TestCreateGrantDuplicate(t *testing.T) {
	db := setup(t)
	createGrants(t, db, admin, view, edit)

	_, err := CreateGrant(db, &admin)
	require.EqualError(t, err, "UNIQUE constraint failed: grants.id")
}

func TestCreateOrUpdateGrantCreate(t *testing.T) {
	db := setup(t)

	grant, err := CreateOrUpdateGrant(db, &admin, &admin)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, grant.ID)
	require.Equal(t, models.GrantKindKubernetes, grant.Kind)
	require.Equal(t, models.GrantKubernetesKindRole, grant.Kubernetes.Kind)
	require.Equal(t, "admin", grant.Kubernetes.Name)
}

func TestCreateOrUpdateGrantUpdateKubernetes(t *testing.T) {
	db := setup(t)
	createGrants(t, db, admin, view, edit)

	clusterAdmin := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: "cluster-role",
			Name: "cluster-admin",
		},
	}

	grant, err := CreateOrUpdateGrant(db, &clusterAdmin, &admin)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, grant.ID)
	require.Equal(t, models.GrantKindKubernetes, grant.Kind)
	require.Equal(t, models.GrantKubernetesKindClusterRole, grant.Kubernetes.Kind)
	require.Equal(t, "cluster-admin", grant.Kubernetes.Name)

	fromDB, err := GetGrant(db, &clusterAdmin)
	require.NoError(t, err)
	require.Equal(t, models.GrantKubernetesKindClusterRole, fromDB.Kubernetes.Kind)
	require.Equal(t, "cluster-admin", fromDB.Kubernetes.Name)
}

func TestGetGrant(t *testing.T) {
	db := setup(t)
	createGrants(t, db, admin, view, edit)

	grant, err := GetGrant(db, &models.Grant{Kind: "kubernetes"})
	require.NoError(t, err)
	require.Equal(t, models.GrantKindKubernetes, grant.Kind)
}

func TestGetGrantGrantSelector(t *testing.T) {
	db := setup(t)
	createGrants(t, db, admin, view, edit)

	grant, err := GetGrant(db, GrantSelector(db, &view))
	require.NoError(t, err)
	require.Equal(t, models.GrantKindKubernetes, grant.Kind)
	require.Equal(t, models.GrantKubernetesKindRole, grant.Kubernetes.Kind)
	require.Equal(t, "view", grant.Kubernetes.Name)
}

func TestGetGrantStrictGrantSelector(t *testing.T) {
	db := setup(t)

	namespaced := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind:      "role",
			Name:      "edit",
			Namespace: "infrahq",
		},
	}

	_, err := CreateGrant(db, &namespaced)
	require.NoError(t, err)

	partial := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind:      "role",
			Name:      "edit",
			Namespace: "",
		},
	}

	_, err = GetGrant(db, StrictGrantSelector(db, &partial))
	require.EqualError(t, err, "record not found")
}

func TestGetGrantGrantSelectorByDestination(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop)

	destination, err := GetDestination(db, &destinationDevelop)
	require.NoError(t, err)

	namespaced := models.Grant{
		Destination: *destination,
		Kind:        models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind:      "role",
			Name:      "edit",
			Namespace: "infrahq",
		},
	}

	_, err = CreateGrant(db, &namespaced)
	require.NoError(t, err)

	partial := models.Grant{
		Destination: *destination,
	}

	grant, err := GetGrant(db, GrantSelector(db, &partial))
	require.NoError(t, err)
	require.Equal(t, destination.ID, grant.DestinationID)
	require.Equal(t, models.GrantKindKubernetes, grant.Kind)
	require.Equal(t, models.GrantKubernetesKindRole, grant.Kubernetes.Kind)
	require.Equal(t, "edit", grant.Kubernetes.Name)
}

func TestListGrants(t *testing.T) {
	db := setup(t)
	createGrants(t, db, admin, view, edit)

	grants, err := ListGrants(db, &models.Grant{})
	require.NoError(t, err)
	require.Len(t, grants, 3)

	grants, err = ListGrants(db, &models.Grant{Kind: "nonexistent"})
	require.NoError(t, err)
	require.Len(t, grants, 0)
}

func TestListGrantsGrantSelector(t *testing.T) {
	db := setup(t)
	createGrants(t, db, admin, view, edit)

	grant := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Name: "edit",
		},
	}

	grants, err := ListGrants(db, GrantSelector(db, &grant))
	require.NoError(t, err)
	require.Len(t, grants, 1)
}

func TestDeleteGrants(t *testing.T) {
	db := setup(t)
	createGrants(t, db, admin, view, edit)

	partial := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Name: "edit",
		},
	}

	_, err := GetGrant(db, GrantSelector(db, &partial))
	require.NoError(t, err)

	err = DeleteGrants(db, GrantSelector(db, &partial))
	require.NoError(t, err)

	_, err = GetGrant(db, GrantSelector(db, &partial))
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent grant should not fail
	err = DeleteGrants(db, GrantSelector(db, &partial))
	require.NoError(t, err)

	// deleting a grant should not delete unrelated grants
	grants, err := ListGrants(db, &models.Grant{})
	require.NoError(t, err)
	require.Len(t, grants, 2)
}
