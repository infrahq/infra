package authn

import (
	"context"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

type AuthenticatedIdentity struct {
	Identity      *models.Identity
	Provider      *models.Provider
	SessionExpiry time.Time
	AuthScope     AuthScope
	// CredentialUpdateRequired indicates that the login used credentials that
	// must be updated because they will no longer be valid after this login.
	CredentialUpdateRequired bool
}

type LoginMethod interface {
	Authenticate(ctx context.Context, db *data.Transaction, requestedExpiry time.Time) (AuthenticatedIdentity, error)
	Name() string // Name returns the name of the authentication method used
}

type AuthScope struct {
	PasswordResetOnly bool
}

type LoginResult struct {
	AccessKey                *models.AccessKey
	Bearer                   string
	User                     *models.Identity
	CredentialUpdateRequired bool
	OrganizationName         string
}

func Login(
	ctx context.Context,
	db *data.Transaction,
	loginMethod LoginMethod,
	requestedExpiry time.Time,
	inactivityTimeout time.Duration,
) (LoginResult, error) {
	// challenge the user to authenticate
	authenticated, err := loginMethod.Authenticate(ctx, db, requestedExpiry)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to login: %w", err)
	}

	// login authentication was successful, create an access key for the user

	accessKey := &models.AccessKey{
		IssuedFor:           authenticated.Identity.ID,
		IssuedForName:       authenticated.Identity.Name,
		ProviderID:          authenticated.Provider.ID,
		ExpiresAt:           authenticated.SessionExpiry,
		InactivityTimeout:   time.Now().UTC().Add(inactivityTimeout),
		InactivityExtension: inactivityTimeout,
		Scopes: models.CommaSeparatedStrings{
			models.ScopeAllowCreateAccessKey,
			models.ScopeAllowApproveDeviceFlowRequest,
		},
	}

	if authenticated.AuthScope.PasswordResetOnly {
		accessKey.Scopes = append(accessKey.Scopes, models.ScopePasswordReset)
	}

	bearer, err := data.CreateAccessKey(db, accessKey)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to create access key after login: %w", err)
	}

	authenticated.Identity.LastSeenAt = time.Now().UTC()
	if err := data.UpdateIdentity(db, authenticated.Identity); err != nil {
		return LoginResult{}, fmt.Errorf("login failed to update last seen: %w", err)
	}

	org, err := data.GetOrganization(db, data.GetOrganizationOptions{ByID: accessKey.OrganizationID})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessKey:                accessKey,
		Bearer:                   bearer,
		User:                     authenticated.Identity,
		CredentialUpdateRequired: authenticated.CredentialUpdateRequired,
		OrganizationName:         org.Name,
	}, nil
}
