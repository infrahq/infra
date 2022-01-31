package data

import (
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

func ListMachines(db *gorm.DB, selectors ...SelectorFunc) ([]models.Machine, error) {
	machines := make([]models.Machine, 0)
	if err := list(db, &models.Machine{}, &machines, selectors); err != nil {
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

func DeleteMachine(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListMachines(db, selectors...)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, m := range toDelete {
			ids = append(ids, m.ID)
		}

		return remove(db, &models.Machine{}, ids)
	}

	return internal.ErrNotFound
}
