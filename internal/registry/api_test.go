package registry

import (
	"errors"

	"gorm.io/gorm"
)

func addUser(db *gorm.DB, email string, password string, admin bool) (tokenId string, tokenSecret string, err error) {
	var token Token
	var secret string
	err = db.Transaction(func(tx *gorm.DB) error {
		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}
		var user User

		err := infraSource.CreateUser(tx, &user, email, password, admin)
		if err != nil {
			return err
		}

		secret, err = NewToken(tx, user.Id, &token)
		if err != nil {
			return errors.New("could not create token")
		}

		return nil
	})
	if err != nil {
		return "", "", err
	}

	return token.Id, secret, nil
}
