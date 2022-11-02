package data

import (
	"errors"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreatePasswordResetToken(db GormTxn, user *models.Identity, ttl time.Duration) (*models.PasswordResetToken, error) {
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
		ExpiresAt:  time.Now().Add(ttl).UTC(),
	}

	tries++
	if err = save(db, prt); err != nil {
		// TODO: must use errors.As for error types
		if tries <= 3 && errors.Is(err, UniqueConstraintError{}) {
			logging.Warnf("generated random token %q already exists in the database", token)
			goto retry // on the off chance the token exists.
		}
		return nil, err
	}

	return prt, nil
}

func GetUserIDForPasswordResetToken(tx WriteTxn, token string) (uid.ID, error) {
	stmt := `
		DELETE from password_reset_tokens
		WHERE token = ? AND organization_id = ?
		RETURNING identity_id, expires_at`

	var userID uid.ID
	var expiresAt time.Time
	err := tx.QueryRow(stmt, token, tx.OrganizationID()).Scan(&userID, &expiresAt)
	if err != nil {
		return 0, handleError(err)
	}

	if expiresAt.Before(time.Now()) {
		return 0, internal.ErrExpired
	}
	return userID, nil
}
