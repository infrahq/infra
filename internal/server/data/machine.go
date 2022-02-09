package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateMachine(db *gorm.DB, machine *models.Machine) error {
	return add(db, machine)
}

func ListMachines(db *gorm.DB, selectors ...SelectorFunc) ([]models.Machine, error) {
	return list[models.Machine](db, selectors...)
}

func GetMachine(db *gorm.DB, selectors ...SelectorFunc) (*models.Machine, error) {
	return get[models.Machine](db, selectors...)
}

func DeleteMachine(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := list[models.Machine](db, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, m := range toDelete {
		ids = append(ids, m.ID)
	}

	return deleteAll[models.Machine](db, ByIDs(ids))
}
