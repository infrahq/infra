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

func CreateProvider(c *gin.Context, provider *models.Provider) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
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

	return data.ListProviders(db, data.ByOptionalName(name))
}

func SaveProvider(c *gin.Context, provider *models.Provider) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.SaveProvider(db, provider)
}

func DeleteProvider(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.DeleteProviders(db, data.ByID(id))
}

// RetrieveUserProviderTokens gets the providerUser for the current session token was created for
func RetrieveUserProviderTokens(c *gin.Context) (*models.ProviderUser, error) {
	// added by the authentication middleware
	identity := CurrentIdentity(c)
	if identity == nil {
		return nil, errors.New("no provider token context user")
	}

	// does not need authorization check, this action is limited to the calling user
	db := getDB(c)

	accessKey := currentAccessKey(c)

	return data.GetProviderUser(db, accessKey.ProviderID, identity.ID)
}

func InfraProvider(c *gin.Context) *models.Provider {
	db := getDB(c)

	return data.InfraProvider(db)
}

func ExchangeAuthCodeForAccessKey(c *gin.Context, code string, provider *models.Provider, oidc authn.OIDC, expires time.Time, redirectURL string) (*models.Identity, string, error) {
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

	user, err := data.GetIdentity(db.Preload("Groups"), data.ByName(email))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, "", fmt.Errorf("get user: %w", err)
		}

		user = &models.Identity{Name: email, Kind: models.UserKind}

		if err := data.CreateIdentity(db, user); err != nil {
			return nil, "", fmt.Errorf("create user: %w", err)
		}

		// by default the user role in infra can see all destinations
		// #1084 - create grants for only destinations a user has access to
		roleGrant := &models.Grant{Subject: user.PolyID(), Privilege: models.InfraUserRole, Resource: "infra"}
		if err := data.CreateGrant(db, roleGrant); err != nil {
			return nil, "", fmt.Errorf("user role grant: %w", err)
		}
	}

	providerUser, err := data.CreateProviderUser(db, provider, user)
	if err != nil {
		return nil, "", fmt.Errorf("add user for provider login: %w", err)
	}

	providerUser.RedirectURL = redirectURL
	providerUser.AccessToken = models.EncryptedAtRest(accessToken)
	providerUser.RefreshToken = models.EncryptedAtRest(refreshToken)
	providerUser.ExpiresAt = expiry
	err = data.UpdateProviderUser(db, providerUser)
	if err != nil {
		return nil, "", err
	}

	// get current identity provider groups
	info, err := oidc.GetUserInfo(providerUser)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, "", fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return nil, "", fmt.Errorf("login user info: %w", err)
	}

	err = UpdateUserInfoFromProvider(c, info, user, provider)
	if err != nil {
		return nil, "", fmt.Errorf("update info on login: %w", err)
	}

	key := &models.AccessKey{
		IssuedFor:  user.ID,
		ProviderID: provider.ID,
		ExpiresAt:  expires,
	}

	body, err := data.CreateAccessKey(db, key)
	if err != nil {
		return nil, body, fmt.Errorf("create access key: %w", err)
	}

	user.LastSeenAt = time.Now().UTC()
	if err := data.SaveIdentity(db, user); err != nil {
		return nil, "", fmt.Errorf("login update last seen: %w", err)
	}

	return user, body, nil
}
