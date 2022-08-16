package authn

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
)

// keyExchangeAuthn allows exchanging a valid access key for new access key with a shorter lifetime
type keyExchangeAuthn struct {
	RequestingAccessKey string // the access key being presented in the login request
}

func NewKeyExchangeAuthentication(requestingAccessKey string) LoginMethod {
	return &keyExchangeAuthn{
		RequestingAccessKey: requestingAccessKey,
	}
}

func (a *keyExchangeAuthn) Authenticate(_ context.Context, db *gorm.DB, requestedExpiry time.Time) (AuthenticatedIdentity, error) {
	validatedRequestKey, err := data.ValidateAccessKey(db, a.RequestingAccessKey)
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("invalid access key in exchange: %w", err)
	}

	sessionExpiry := requestedExpiry

	if sessionExpiry.After(validatedRequestKey.ExpiresAt) {
		logging.L.Trace().Msg("key exchanged with expiry before default, set exchanged key expiry to match requesting key")
		sessionExpiry = validatedRequestKey.ExpiresAt
	}

	identity, err := data.GetIdentity(db, data.ByID(validatedRequestKey.IssuedFor))
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("user is not valid: %w", err) // the user was probably deleted
	}

	return AuthenticatedIdentity{
		Identity:      identity,
		Provider:      data.InfraProvider(db),
		SessionExpiry: sessionExpiry,
	}, nil
}

func (a *keyExchangeAuthn) Name() string {
	return "exchange"
}

func (a *keyExchangeAuthn) RequiresUpdate(db *gorm.DB) (bool, error) {
	return false, nil // not applicable to key exchange
}
