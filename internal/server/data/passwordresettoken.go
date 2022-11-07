package data

import (
	"errors"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type passwordResetToken struct {
	ID uid.ID
	models.OrganizationMember

	Token      string
	IdentityID uid.ID
	ExpiresAt  time.Time
}

func (passwordResetToken) Table() string {
	return "password_reset_tokens"
}

func (p passwordResetToken) Columns() []string {
	return []string{"expires_at", "id", "identity_id", "organization_id", "token"}
}

func (p passwordResetToken) Values() []any {
	return []any{p.ExpiresAt, p.ID, p.IdentityID, p.OrganizationID, p.Token}
}

func (p *passwordResetToken) ScanFields() []any {
	return []any{&p.ExpiresAt, &p.ID, &p.IdentityID, &p.OrganizationID, &p.Token}
}

func (p *passwordResetToken) OnInsert() error {
	return nil
}

func CreatePasswordResetToken(tx WriteTxn, userID uid.ID, ttl time.Duration) (string, error) {
	if userID == 0 || ttl == 0 {
		return "", fmt.Errorf("a userID and ttl are required")
	}

	tries := 0
	var ucErr UniqueConstraintError

retry:
	token, err := generate.CryptoRandom(10, generate.CharsetAlphaNumeric)
	if err != nil {
		return "", err
	}

	prt := &passwordResetToken{
		ID:         uid.New(),
		Token:      token,
		IdentityID: userID,
		ExpiresAt:  time.Now().Add(ttl).UTC(),
	}

	tries++
	if err = insert(tx, prt); err != nil {
		if tries <= 3 && errors.As(err, &ucErr) {
			logging.Warnf("generated random token %q already exists in the database", token)
			goto retry // on the off chance the token exists.
		}
		return "", err
	}

	return prt.Token, nil
}

// ClaimPasswordResetToken deletes the password reset token, and returns the
// user ID that was associated with the token. Returns an error if the token
// does not exist or has expired.
func ClaimPasswordResetToken(tx WriteTxn, token string) (uid.ID, error) {
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
