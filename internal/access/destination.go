package access

import (
	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/uuid"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

const (
	PermissionDestination       Permission = "infra.destination.*"
	PermissionDestinationCreate Permission = "infra.destination.create"
	PermissionDestinationRead   Permission = "infra.destination.read"
	PermissionDestinationUpdate Permission = "infra.destination.update"
	PermissionDestinationDelete Permission = "infra.destination.delete"
)

func CreateDestination(c *gin.Context, destination *models.Destination) error {
	db, err := requireAuthorization(c, PermissionDestinationCreate)
	if err != nil {
		return err
	}

	return data.CreateDestination(db, destination)
}

func UpdateDestination(c *gin.Context, destination *models.Destination) error {
	db, err := requireAuthorization(c, PermissionDestinationUpdate)
	if err != nil {
		return err
	}

	return data.UpdateDestination(db, destination)
}

func GetDestination(c *gin.Context, id uuid.UUID) (*models.Destination, error) {
	db, err := requireAuthorization(c, PermissionDestinationRead)
	if err != nil {
		return nil, err
	}

	return data.GetDestination(db, data.ByID(id))
}

func ListDestinations(c *gin.Context, kind, nodeID, name string, labels []string) ([]models.Destination, error) {
	db, err := requireAuthorization(c, PermissionDestinationRead)
	if err != nil {
		return nil, err
	}

	return data.ListDestinations(db, db.Where(
		data.LabelSelector(db, "destination_id", labels...),
		db.Where(
			&models.Destination{
				Kind:   models.DestinationKind(kind),
				NodeID: nodeID,
				Name:   name,
			}),
	))
}

func DeleteDestination(c *gin.Context, id uuid.UUID) error {
	db, err := requireAuthorization(c, PermissionDestinationDelete)
	if err != nil {
		return err
	}

	return data.DeleteDestinations(db, data.ByID(id))
}

func ListUserDestinations(c *gin.Context, userID uuid.UUID) ([]models.Destination, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionDestinationRead, func(user *models.User) bool {
		return userID == user.ID
	})
	if err != nil {
		return nil, err
	}

	return data.ListUserDestinations(db, userID)
}
