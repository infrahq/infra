package access

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

const (
	PermissionRole       Permission = "infra.role.*"
	PermissionRoleCreate Permission = "infra.role.create"
	PermissionRoleRead   Permission = "infra.role.read"
	PermissionRoleUpdate Permission = "infra.role.update"
	PermissionRoleDelete Permission = "infra.role.delete"
)

func GetRole(c *gin.Context, id string) (*models.Role, error) {
	db, err := RequireAuthorization(c, PermissionRoleRead)
	if err != nil {
		return nil, err
	}

	role, err := models.NewRole(id)
	if err != nil {
		return nil, err
	}

	return data.GetRole(db, role)
}

func ListRoles(c *gin.Context, name, kind, destinationID string) ([]models.Role, error) {
	db, err := RequireAuthorization(c, PermissionRoleRead)
	if err != nil {
		return nil, err
	}

	// hardcode role kind to Kubernetes for now
	role := models.Role{
		Kind: models.RoleKindKubernetes,
	}

	switch role.Kind {
	case models.RoleKindKubernetes:
		role.Kubernetes.Kind = models.RoleKubernetesKind(kind)
		role.Kubernetes.Name = name
	}

	if id, err := uuid.Parse(destinationID); err == nil {
		role.DestinationID = id
	}

	return data.ListRoles(db, data.RoleSelector(db, &role))
}
