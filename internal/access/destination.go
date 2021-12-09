package access

import (
	"github.com/gin-gonic/gin"

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

func CreateDestination(c *gin.Context, destination *models.Destination) (*models.Destination, error) {
	db, err := RequireAuthorization(c, PermissionDestinationCreate)
	if err != nil {
		return nil, err
	}

	return data.CreateOrUpdateDestination(db, destination, &models.Destination{NodeID: destination.NodeID})
}

func GetDestination(c *gin.Context, id string) (*models.Destination, error) {
	db, err := RequireAuthorization(c, PermissionDestinationRead)
	if err != nil {
		return nil, err
	}

	destination, err := models.NewDestination(id)
	if err != nil {
		return nil, err
	}

	return data.GetDestination(db, destination)
}

func ListDestinations(c *gin.Context, kind, nodeID, name string, labels []string) ([]models.Destination, error) {
	db, err := RequireAuthorization(c, PermissionDestinationRead)
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

func UpdateDestination(c *gin.Context, id string, destination *models.Destination) (*models.Destination, error) {
	db, err := RequireAuthorization(c, PermissionDestinationUpdate)
	if err != nil {
		return nil, err
	}

	return data.UpdateDestination(db, destination, data.ByID(id))
}

func DeleteDestination(c *gin.Context, id string) error {
	db, err := RequireAuthorization(c, PermissionDestinationDelete)
	if err != nil {
		return err
	}

	return data.DeleteDestinations(db, data.ByID(id))
}
