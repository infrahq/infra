package access

import (
	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
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

func SaveDestination(c *gin.Context, destination *models.Destination) error {
	db, err := requireAuthorization(c, PermissionDestinationUpdate)
	if err != nil {
		return err
	}

	return data.SaveDestination(db, destination)
}

func GetDestination(c *gin.Context, id uid.ID) (*models.Destination, error) {
	db, err := requireAuthorization(c, PermissionDestinationRead)
	if err != nil {
		return nil, err
	}

	return data.GetDestination(db, data.ByID(id))
}

func ListDestinations(c *gin.Context, uniqueID, name string) ([]models.Destination, error) {
	db, err := requireAuthorization(c, PermissionDestinationRead)
	if err != nil {
		return nil, err
	}

	return data.ListDestinations(db, data.ByUniqueID(uniqueID), data.ByName(name))
}

func DeleteDestination(c *gin.Context, id uid.ID) error {
	db, err := requireAuthorization(c, PermissionDestinationDelete)
	if err != nil {
		return err
	}

	return data.DeleteDestinations(db, data.ByID(id))
}
