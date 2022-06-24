package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateCredential(db *gorm.DB, credential *models.Credential) error {
	return add(db, credential)
}

func SaveCredential(db *gorm.DB, credential *models.Credential) error {
	return save(db, credential)
}

func GetCredential(db *gorm.DB, selectors ...SelectorFunc) (*models.Credential, error) {
	return get[models.Credential](db, selectors...)
}

func DeleteCredential(db *gorm.DB, id uid.ID) error {
	return delete[models.Credential](db, id)
}
