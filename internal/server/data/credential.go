package data

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func validateCredential(c *models.Credential) error {
	if len(c.PasswordHash) == 0 {
		return fmt.Errorf("passwordHash is required")
	}
	return nil
}

func CreateCredential(db *gorm.DB, credential *models.Credential) error {
	if err := validateCredential(credential); err != nil {
		return err
	}
	return add(db, credential)
}

func SaveCredential(db *gorm.DB, credential *models.Credential) error {
	if err := validateCredential(credential); err != nil {
		return err
	}
	return save(db, credential)
}

func GetCredential(db *gorm.DB, selectors ...SelectorFunc) (*models.Credential, error) {
	return get[models.Credential](db, selectors...)
}

func DeleteCredential(db *gorm.DB, id uid.ID) error {
	return delete[models.Credential](db, id)
}
