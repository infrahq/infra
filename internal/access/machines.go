package access

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionMachineCreate Permission = "infra.machine.create"
	PermissionMachineRead   Permission = "infra.machine.read"
	PermissionMachineDelete Permission = "infra.machine.delete"
)

func CreateMachine(c *gin.Context, machine *models.Machine) error {
	db, err := requireAuthorization(c, PermissionMachineCreate)
	if err != nil {
		return err
	}

	// do not let a caller create a machine with more permissions than they have
	permissions, ok := c.MustGet("permissions").(string)
	if !ok {
		// there should have been permissions set by this point
		return internal.ErrForbidden
	}

	if !AllRequired(strings.Split(permissions, " "), strings.Split(machine.Permissions, " ")) {
		return fmt.Errorf("cannot create a machine identity with permissions not granted to the creator")
	}

	if err := data.CreateMachine(db, machine); err != nil {
		return fmt.Errorf("create machine: %w", err)
	}

	return nil
}

func ListMachines(c *gin.Context, name string) ([]models.Machine, error) {
	db, err := requireAuthorization(c, PermissionMachineRead)
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
	db, err := requireAuthorization(c, PermissionMachineDelete)
	if err != nil {
		return err
	}

	return data.DeleteMachine(db, data.ByID(id))
}
