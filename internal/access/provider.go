package access

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// CurrentIdentityProvider returns the provider for the current identity in the request context
func CurrentIdentityProvider(c *gin.Context) (*models.Provider, error) {
	u, ok := c.MustGet("user").(*models.User)
	if !ok {
		return nil, fmt.Errorf("no user in request context")
	}

	return GetProvider(c, u.ProviderID)
}

func CreateProvider(c *gin.Context, provider *models.Provider) error {
	db, err := requireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.CreateProvider(db, provider)
}

func GetProvider(c *gin.Context, id uid.ID) (*models.Provider, error) {
	db := getDB(c)

	return data.GetProvider(db, data.ByID(id))
}

func ListProviders(c *gin.Context, name string) ([]models.Provider, error) {
	db := getDB(c)

	return data.ListProviders(db, data.ByName(name))
}

func SaveProvider(c *gin.Context, provider *models.Provider) error {
	db, err := requireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.SaveProvider(db, provider)
}

func DeleteProvider(c *gin.Context, id uid.ID) error {
	db, err := requireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.DeleteProviders(db, data.ByID(id))
}

// RetrieveUserProviderTokens gets the provider tokens that the current session token was created for
func RetrieveUserProviderTokens(c *gin.Context) (*models.ProviderToken, error) {
	// added by the authentication middleware
	user, ok := c.MustGet("user").(*models.User)
	if !ok {
		return nil, errors.New("no provider token context user")
	}

	// does not need authorization check, this action is limited to the calling user
	db := getDB(c)

	return data.GetProviderToken(db, data.ByUserID(user.ID))
}

// UpdateProviderToken overwrites an existing set of provider tokens
func UpdateProviderToken(c *gin.Context, providerToken *models.ProviderToken) error {
	// does not need authorization check, this function should only be called internally
	db := getDB(c)

	return data.UpdateProviderToken(db, providerToken)
}

func ExchangeAuthCodeForAccessKey(c *gin.Context, code string, provider *models.Provider, oidc authn.OIDC, sessionDuration time.Duration, redirectURL string) (*models.User, string, error) {
	// does not need authorization check, this function should only be called internally
	db := getDB(c)

	// exchange code for tokens from identity provider (these tokens are for the IDP, not Infra)
	accessToken, refreshToken, expiry, email, err := oidc.ExchangeAuthCodeForProviderTokens(code)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, "", fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return nil, "", fmt.Errorf("exhange code for tokens: %w", err)
	}

	user, err := data.GetUser(db, data.ByEmail(email))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, "", fmt.Errorf("get user: %w", err)
		}

		user = &models.User{Email: email}

		if err := data.CreateUser(db, user); err != nil {
			return nil, "", fmt.Errorf("create user: %w", err)
		}

		// by default the user role in infra can see all destinations
		// #1084 - create grants for only destinations a user has access to
		roleGrant := &models.Grant{Identity: user.PolymorphicIdentifier(), Privilege: models.InfraUserRole, Resource: "infra"}
		if err := data.CreateGrant(db, roleGrant); err != nil {
			return nil, "", fmt.Errorf("user role grant: %w", err)
		}
	}

	err = data.AppendProviderUsers(db, provider, user)
	if err != nil {
		return nil, "", fmt.Errorf("add user for provider login: %w", err)
	}

	provToken := &models.ProviderToken{
		UserID:       user.ID,
		ProviderID:   provider.ID,
		RedirectURL:  redirectURL,
		AccessToken:  models.EncryptedAtRest(accessToken),
		RefreshToken: models.EncryptedAtRest(refreshToken),
		ExpiresAt:    expiry,
	}

	// create or update the provider token for this user, one set of tokens/user for each provider
	existing, err := data.GetProviderToken(db, data.ByUserID(user.ID))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, "", fmt.Errorf("existing provider token: %w", err)
		}

		if err := data.CreateProviderToken(db, provToken); err != nil {
			return nil, "", fmt.Errorf("create provider tokens: %w", err)
		}
	} else {
		if existing.ProviderID != provToken.ProviderID {
			// revoke the users current session token, their grants may be about to change
			if err := data.DeleteAccessKeys(db, data.ByUserID(user.ID)); err != nil && !errors.Is(err, internal.ErrNotFound) {
				return nil, "", fmt.Errorf("revoke old session token: %w", err)
			}
		}

		provToken.ID = existing.ID

		if err := data.UpdateProviderToken(db, provToken); err != nil {
			return nil, "", fmt.Errorf("update provider token: %w", err)
		}
	}

	// get current identity provider groups
	info, err := oidc.GetUserInfo(provToken)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, "", fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return nil, "", fmt.Errorf("login user info: %w", err)
	}

	err = UpdateUserInfo(c, info, user, provider)
	if err != nil {
		return nil, "", fmt.Errorf("update info on login: %w", err)
	}

	key := &models.AccessKey{
		IssuedFor: user.PolymorphicIdentifier(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	body, err := data.CreateAccessKey(db, key)
	if err != nil {
		return nil, body, fmt.Errorf("create access key: %w", err)
	}

	user.LastSeenAt = time.Now()
	if err := data.SaveUser(db, user); err != nil {
		return nil, "", fmt.Errorf("login update last seen: %w", err)
	}

	return user, body, nil
}
