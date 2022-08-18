package data

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreatePasswordResetToken(db *gorm.DB, user *models.Identity) (*models.PasswordResetToken, error) {
	tries := 0
retry:
	token, err := generate.CryptoRandom(10, generate.CharsetAlphaNumeric)
	if err != nil {
		return nil, err
	}

	prt := &models.PasswordResetToken{
		ID:         uid.New(),
		Token:      token,
		IdentityID: user.ID,
		ExpiresAt:  time.Now().Add(72 * time.Hour).UTC(),
	}

	tries++
	if err = save(db, prt); err != nil {
		if tries <= 3 && errors.Is(err, UniqueConstraintError{}) {
			logging.Warnf("generated random token %q already exists in the database", token)
			goto retry // on the off chance the token exists.
		}
		return nil, err
	}

	return prt, nil
}

func GetPasswordResetTokenByToken(db *gorm.DB, token string) (*models.PasswordResetToken, error) {
	prts, err := list[models.PasswordResetToken](db, &models.Pagination{Limit: 1}, func(db *gorm.DB) *gorm.DB {
		return db.Where("token = ?", token)
	})
	if err != nil {
		return nil, err
	}

	if len(prts) != 1 {
		return nil, internal.ErrNotFound
	}

	if prts[0].ExpiresAt.Before(time.Now()) {
		_ = DeletePasswordResetToken(db, &prts[0])
		return nil, internal.ErrExpired
	}

	return &prts[0], nil
}

func DeletePasswordResetToken(db *gorm.DB, prt *models.PasswordResetToken) error {
	return delete[models.PasswordResetToken](db, prt.ID)
}
