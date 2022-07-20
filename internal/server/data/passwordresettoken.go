package data

import (
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
)

func CreatePasswordResetToken(db *gorm.DB, user *models.Identity) (*models.PasswordResetToken, error) {
	token, err := generate.CryptoRandom(10, generate.CharsetAlphaNumeric)
	if err != nil {
		return nil, err
	}

	prt := &models.PasswordResetToken{
		Token:      token,
		IdentityID: user.ID,
		ExpiresAt:  time.Now().Add(10 * time.Minute).UTC(),
	}

	if err = save(db, prt); err != nil {
		return nil, err
	}

	return prt, nil
}

func GetPasswordResetTokenByToken(db *gorm.DB, token string) (*models.PasswordResetToken, error) {
	prts, err := list[models.PasswordResetToken](db, &models.Pagination{Limit: 1}, func(db *gorm.DB) *gorm.DB {
		return db.Where("token = ? and expires_at > ?", token, time.Now().UTC())
	})
	if err != nil {
		return nil, err
	}

	if len(prts) != 1 {
		return nil, internal.ErrNotFound
	}

	if prts[0].ExpiresAt.Before(time.Now()) {
		return nil, internal.ErrNotFound
	}

	return &prts[0], nil
}
