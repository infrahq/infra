package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
)

var (
	admin = models.Role{Kind: "kubernetes", Kubernetes: models.RoleKubernetes{Kind: "role", Name: "admin"}}
	view  = models.Role{Kind: "kubernetes", Kubernetes: models.RoleKubernetes{Kind: "role", Name: "view"}}
	edit  = models.Role{Kind: "kubernetes", Kubernetes: models.RoleKubernetes{Kind: "role", Name: "edit"}}
)

func TestRole(t *testing.T) {
	db := setup(t)

	err := db.Create(&admin).Error
	require.NoError(t, err)

	var role models.Role
	err = db.Preload("Kubernetes").First(&role, &models.Role{Kind: "kubernetes"}).Error
	require.NoError(t, err)
	require.Equal(t, models.RoleKindKubernetes, role.Kind)
	require.Equal(t, models.RoleKubernetesKindRole, role.Kubernetes.Kind)
	require.Equal(t, "admin", role.Kubernetes.Name)
}

func TestCreateRole(t *testing.T) {
	db := setup(t)

	role, err := CreateRole(db, &admin)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, role.ID)
	require.Equal(t, admin.Kind, role.Kind)
	require.Equal(t, admin.Kubernetes.Kind, role.Kubernetes.Kind)
	require.Equal(t, admin.Kubernetes.Name, role.Kubernetes.Name)
}

func createRoles(t *testing.T, db *gorm.DB, roles ...models.Role) {
	for i := range roles {
		_, err := CreateRole(db, &roles[i])
		require.NoError(t, err)
	}
}

func TestCreateRoleDuplicate(t *testing.T) {
	db := setup(t)
	createRoles(t, db, admin, view, edit)

	_, err := CreateRole(db, &admin)
	require.EqualError(t, err, "UNIQUE constraint failed: roles.id")
}

func TestCreateOrUpdateRoleCreate(t *testing.T) {
	db := setup(t)

	role, err := CreateOrUpdateRole(db, &admin, &admin)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, role.ID)
	require.Equal(t, models.RoleKindKubernetes, role.Kind)
	require.Equal(t, models.RoleKubernetesKindRole, role.Kubernetes.Kind)
	require.Equal(t, "admin", role.Kubernetes.Name)
}

func TestCreateOrUpdateRoleUpdateKubernetes(t *testing.T) {
	db := setup(t)
	createRoles(t, db, admin, view, edit)

	clusterAdmin := models.Role{
		Kind: models.RoleKindKubernetes,
		Kubernetes: models.RoleKubernetes{
			Kind: "cluster-role",
			Name: "cluster-admin",
		},
	}

	role, err := CreateOrUpdateRole(db, &clusterAdmin, &admin)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, role.ID)
	require.Equal(t, models.RoleKindKubernetes, role.Kind)
	require.Equal(t, models.RoleKubernetesKindClusterRole, role.Kubernetes.Kind)
	require.Equal(t, "cluster-admin", role.Kubernetes.Name)

	fromDB, err := GetRole(db, &clusterAdmin)
	require.NoError(t, err)
	require.Equal(t, models.RoleKubernetesKindClusterRole, fromDB.Kubernetes.Kind)
	require.Equal(t, "cluster-admin", fromDB.Kubernetes.Name)
}

func TestGetRole(t *testing.T) {
	db := setup(t)
	createRoles(t, db, admin, view, edit)

	role, err := GetRole(db, &models.Role{Kind: "kubernetes"})
	require.NoError(t, err)
	require.Equal(t, models.RoleKindKubernetes, role.Kind)
}

func TestGetRoleRoleSelector(t *testing.T) {
	db := setup(t)
	createRoles(t, db, admin, view, edit)

	role, err := GetRole(db, RoleSelector(db, &view))
	require.NoError(t, err)
	require.Equal(t, models.RoleKindKubernetes, role.Kind)
	require.Equal(t, models.RoleKubernetesKindRole, role.Kubernetes.Kind)
	require.Equal(t, "view", role.Kubernetes.Name)
}

func TestGetRoleStrictRoleSelector(t *testing.T) {
	db := setup(t)

	namespaced := models.Role{
		Kind: models.RoleKindKubernetes,
		Kubernetes: models.RoleKubernetes{
			Kind:      "role",
			Name:      "edit",
			Namespace: "infrahq",
		},
	}

	_, err := CreateRole(db, &namespaced)
	require.NoError(t, err)

	partial := models.Role{
		Kind: models.RoleKindKubernetes,
		Kubernetes: models.RoleKubernetes{
			Kind:      "role",
			Name:      "edit",
			Namespace: "",
		},
	}

	_, err = GetRole(db, StrictRoleSelector(db, &partial))
	require.EqualError(t, err, "record not found")
}

func TestGetRoleRoleSelectorByDestination(t *testing.T) {
	db := setup(t)
	createDestinations(t, db, destinationDevelop)

	destination, err := GetDestination(db, &destinationDevelop)
	require.NoError(t, err)

	namespaced := models.Role{
		Destination: *destination,
		Kind:        models.RoleKindKubernetes,
		Kubernetes: models.RoleKubernetes{
			Kind:      "role",
			Name:      "edit",
			Namespace: "infrahq",
		},
	}

	_, err = CreateRole(db, &namespaced)
	require.NoError(t, err)

	partial := models.Role{
		Destination: *destination,
	}

	role, err := GetRole(db, RoleSelector(db, &partial))
	require.NoError(t, err)
	require.Equal(t, destination.ID, role.DestinationID)
	require.Equal(t, models.RoleKindKubernetes, role.Kind)
	require.Equal(t, models.RoleKubernetesKindRole, role.Kubernetes.Kind)
	require.Equal(t, "edit", role.Kubernetes.Name)
}

func TestListRoles(t *testing.T) {
	db := setup(t)
	createRoles(t, db, admin, view, edit)

	roles, err := ListRoles(db, &models.Role{})
	require.NoError(t, err)
	require.Len(t, roles, 3)

	roles, err = ListRoles(db, &models.Role{Kind: "nonexistent"})
	require.NoError(t, err)
	require.Len(t, roles, 0)
}

func TestListRolesRoleSelector(t *testing.T) {
	db := setup(t)
	createRoles(t, db, admin, view, edit)

	role := models.Role{
		Kind: models.RoleKindKubernetes,
		Kubernetes: models.RoleKubernetes{
			Name: "edit",
		},
	}

	roles, err := ListRoles(db, RoleSelector(db, &role))
	require.NoError(t, err)
	require.Len(t, roles, 1)
}

func TestDeleteRoles(t *testing.T) {
	db := setup(t)
	createRoles(t, db, admin, view, edit)

	partial := models.Role{
		Kind: models.RoleKindKubernetes,
		Kubernetes: models.RoleKubernetes{
			Name: "edit",
		},
	}

	_, err := GetRole(db, RoleSelector(db, &partial))
	require.NoError(t, err)

	err = DeleteRoles(db, RoleSelector(db, &partial))
	require.NoError(t, err)

	_, err = GetRole(db, RoleSelector(db, &partial))
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent role should not fail
	err = DeleteRoles(db, RoleSelector(db, &partial))
	require.NoError(t, err)

	// deleting a role should not delete unrelated roles
	roles, err := ListRoles(db, &models.Role{})
	require.NoError(t, err)
	require.Len(t, roles, 2)
}
