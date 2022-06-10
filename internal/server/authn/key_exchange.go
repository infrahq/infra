package authn

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

// keyExchangeAuthn allows exchanging a valid access key for new access key with a shorter lifetime
type keyExchangeAuthn struct {
	RequestingAccessKey string    // the access key being presented in the login request
	RequestedExpiry     time.Time // the expiry of the new access key that would be issued on login
}

func NewKeyExchangeAuthentication(requestingAccessKey string, requestedExpiry time.Time) LoginMethod {
	return &keyExchangeAuthn{
		RequestingAccessKey: requestingAccessKey,
		RequestedExpiry:     requestedExpiry,
	}
}

func (a *keyExchangeAuthn) Authenticate(db *gorm.DB) (*models.Identity, *models.Provider, error) {
	validatedRequestKey, err := data.ValidateAccessKey(db, a.RequestingAccessKey)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid access key in exchange: %w", err)
	}

	if a.RequestedExpiry.After(validatedRequestKey.ExpiresAt) {
		return nil, nil, fmt.Errorf("%w: cannot exchange an access key for another access key with a longer lifetime", internal.ErrBadRequest)
	}

	identity, err := data.GetIdentity(db, data.ByID(validatedRequestKey.IssuedFor))
	if err != nil {
		return nil, nil, fmt.Errorf("user is not valid: %w", err) // the user was probably deleted
	}

	return identity, data.InfraProvider(db), nil
}

func (a *keyExchangeAuthn) Name() string {
	return "exchange"
}

func (a *keyExchangeAuthn) RequiresUpdate(db *gorm.DB) (bool, error) {
	return false, nil // not applicable to key exchange
}
