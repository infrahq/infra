package data

import (
	"fmt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

func ValidateCredential(db *gorm.DB, user *models.User, password string) (bool, error) {
	userCredential, err := GetCredential(db, ByIdentity(user.PolymorphicIdentifier()))
	if err != nil {
		return false, fmt.Errorf("validate creds get user: %w", err)
	}

	if userCredential.OneTimePassword && userCredential.OneTimePasswordUsed {
		return false, fmt.Errorf("one time password cannot be used more than once")
	}

	err = bcrypt.CompareHashAndPassword(userCredential.PasswordHash, []byte(password))
	if err != nil {
		return false, fmt.Errorf("password verify: %w", err)
	}

	if userCredential.OneTimePassword {
		userCredential.OneTimePasswordUsed = true
		err = SaveCredential(db, userCredential)
		if err != nil {
			return false, fmt.Errorf("save otp used: %w", err)
		}
	}

	return userCredential.OneTimePassword, nil
}
