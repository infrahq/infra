package data

import (
	"fmt"

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

// IdentityCredentialMustBeUpdated checks if the associated identity has a one time password and that password has been used
func IdentityCredentialMustBeUpdated(db *gorm.DB, user *models.Identity) (bool, error) {
	userCredential, err := GetCredential(db, ByIdentityID(user.ID))
	if err != nil {
		return false, fmt.Errorf("check identity one time password used: %w", err)
	}

	return userCredential.OneTimePassword && userCredential.OneTimePasswordUsed, nil
}
