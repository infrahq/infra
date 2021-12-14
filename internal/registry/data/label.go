package data

import (
	"errors"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func FindOrInitLabels(db *gorm.DB, labels []models.Label) error {
	for i := range labels {
		label, err := GetLabel(db, ByLabelValue(labels[i].Value))
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return err
			}

			continue
		}

		labels[i] = *label
	}

	return nil
}

func GetLabel(db *gorm.DB, selector SelectorFunc) (*models.Label, error) {
	var label models.Label
	if err := get(db, &models.Label{}, &label, selector(db)); err != nil {
		return nil, err
	}

	return &label, nil
}

func ByLabelValue(value string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("value = ?", value)
	}
}
