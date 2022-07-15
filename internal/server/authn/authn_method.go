package authn

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

type LoginMethod interface {
	Authenticate(ctx context.Context, db *gorm.DB) (*models.Organization, *models.Identity, *models.Provider, AuthScope, error)
	Name() string                             // Name returns the name of the authentication method used
	RequiresUpdate(db *gorm.DB) (bool, error) // Temporary way to check for one time password re-use, remove with #1441
}

type AuthScope struct {
	PasswordResetOnly bool
}

func Login(ctx context.Context, db *gorm.DB, loginMethod LoginMethod, keyExpiresAt time.Time, keyExtension time.Duration) (*models.AccessKey, string, error) {
	// challenge the user to authenticate
	org, identity, provider, scope, err := loginMethod.Authenticate(ctx, db)
	if err != nil {
		return nil, "", fmt.Errorf("failed to login: %w", err)
	}

	// login authentication was successful, create an access key for the user

	accessKey := &models.AccessKey{
		IssuedFor:         identity.ID,
		IssuedForIdentity: identity,
		ProviderID:        provider.ID,
		ExpiresAt:         keyExpiresAt,
		ExtensionDeadline: time.Now().UTC().Add(keyExtension),
		Extension:         keyExtension,
	}
	accessKey.OrganizationID = org.ID

	if scope.PasswordResetOnly {
		accessKey.Scopes = append(accessKey.Scopes, models.ScopePasswordReset)
	}

	bearer, err := data.CreateAccessKey(db, accessKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create access key after login: %w", err)
	}

	identity.LastSeenAt = time.Now().UTC()
	if err := data.SaveIdentity(db, identity); err != nil {
		return nil, "", fmt.Errorf("login failed to update last seen: %w", err)
	}

	return accessKey, bearer, nil
}
