package data

import (
	"errors"
	"fmt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
	"gorm.io/gorm"
)

func CreateMachine(db *gorm.DB, machine *models.Machine) error {
	if err := add(db, &models.Machine{}, machine, &models.Machine{}); err != nil {
		return fmt.Errorf("create machine: %w", err)
	}

	return nil
}

func ListMachines(db *gorm.DB, condition interface{}) ([]models.Machine, error) {
	machines := make([]models.Machine, 0)
	if err := list(db, &models.Machine{}, &machines, condition); err != nil {
		return nil, err
	}

	return machines, nil
}

func GetMachine(db *gorm.DB, selector SelectorFunc) (*models.Machine, error) {
	machine := &models.Machine{}
	if err := get(db, &models.Machine{}, machine, selector); err != nil {
		return nil, err
	}

	return machine, nil
}

func DeleteMachine(db *gorm.DB, id uid.ID) error {
	toDelete, err := GetMachine(db, ByID(id))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("delete machine: %w", err)
		}

		return err
	}

	return remove(db, &models.Machine{}, toDelete.ID)
}
