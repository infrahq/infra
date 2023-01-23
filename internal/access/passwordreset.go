package access

import (
	"errors"
	"fmt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

// TODO: remove
func VerifiedPasswordReset(rCtx RequestContext, token, password string) (*models.Identity, error) {
	// no auth required
	tx := rCtx.DBTxn

	userID, err := data.ClaimPasswordResetToken(tx, token)
	if err != nil {
		return nil, err
	}

	user, err := data.GetIdentity(tx, data.GetIdentityOptions{ByID: userID})
	if err != nil {
		return nil, err
	}

	if !user.Verified {
		user.Verified = true
		if err = data.UpdateIdentity(tx, user); err != nil {
			return nil, err
		}
	}

	credential, err := data.GetCredentialByUserID(tx, user.ID)
	switch {
	case errors.Is(err, internal.ErrNotFound):
		if err := createCredential(tx, user, &models.Credential{}, password); err != nil {
			return nil, err
		}

		return user, nil

	case err != nil:
		return nil, fmt.Errorf("get credential: %w", err)
	}

	hash, err := GenerateFromPassword(password)
	if err != nil {
		return nil, err
	}

	credential.OneTimePassword = false
	credential.PasswordHash = hash

	if err := data.UpdateCredential(tx, credential); err != nil {
		return nil, err
	}

	return user, nil
}
