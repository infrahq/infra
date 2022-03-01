package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isMachineSelf is used by authorization checks to see if the calling machine is requesting their own attributes
func isMachineSelf(c *gin.Context, requestedResourceID uid.ID) (bool, error) {
	machine := CurrentMachine(c)
	return machine != nil && machine.ID == requestedResourceID, nil
}

func CurrentMachine(c *gin.Context) *models.Machine {
	machineObj, exists := c.Get("machine")
	if !exists {
		return nil
	}

	machine, ok := machineObj.(*models.Machine)
	if !ok {
		return nil
	}

	return machine
}

func CreateMachine(c *gin.Context, machine *models.Machine) error {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return err
	}

	if err := data.CreateMachine(db, machine); err != nil {
		return fmt.Errorf("create machine: %w", err)
	}

	return nil
}

func GetMachine(c *gin.Context, id uid.ID) (*models.Machine, error) {
	db, err := requireInfraRole(c, AdminRole, ConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.GetMachine(db, data.ByID(id))
}

func ListMachines(c *gin.Context, name string) ([]models.Machine, error) {
	db, err := requireInfraRole(c, AdminRole, ConnectorRole)
	if err != nil {
		return nil, err
	}

	machines, err := data.ListMachines(db, data.ByName(name))
	if err != nil {
		return nil, err
	}

	return machines, nil
}

func DeleteMachine(c *gin.Context, id uid.ID) error {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return err
	}

	return data.DeleteMachine(db, data.ByID(id))
}
