package access

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/registry/authn"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionToken       Permission = "infra.token.*" // nolint:gosec
	PermissionTokenRead   Permission = "infra.token.read"
	PermissionTokenRevoke Permission = "infra.token.revoke" // nolint:gosec

	PermissionAPIToken Permission = "infra.apiToken.*" // nolint:gosec

	PermissionAPITokenCreate Permission = "infra.apiToken.create" // nolint:gosec
	PermissionAPITokenRead   Permission = "infra.apiToken.read"   // nolint:gosec
	PermissionAPITokenDelete Permission = "infra.apiToken.delete" // nolint:gosec

	PermissionCredentialCreate Permission = "infra.credential.create" //nolint:gosec
)

// the default permissions a user is assigned on account creation
var DefaultPermissions = strings.Join([]string{
	string(PermissionUserRead),
	string(PermissionTokenRevoke),
	string(PermissionCredentialCreate),
}, " ")

func ExchangeAuthCodeForSessionToken(c *gin.Context, code string, provider *models.Provider, oidc authn.OIDC, sessionDuration time.Duration) (*models.User, string, error) {
	db, err := requireAuthorization(c)
	if err != nil {
		return nil, "", err
	}

	// exchange code for tokens from identity provider (these tokens are for the IDP, not Infra)
	accessToken, refreshToken, expiry, email, err := oidc.ExchangeAuthCodeForProviderTokens(code)
	if err != nil {
		return nil, "", fmt.Errorf("exhange code for tokens: %w", err)
	}

	user, err := data.GetUser(db, data.ByEmail(email))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, "", fmt.Errorf("get user: %w", err)
		}

		user = &models.User{Email: email, Permissions: DefaultPermissions}
		if err = data.CreateUser(db, user); err != nil {
			return nil, "", fmt.Errorf("create user: %w", err)
		}
	}

	err = data.AppendProviderUsers(db, provider, *user)
	if err != nil {
		return nil, "", fmt.Errorf("add user for provider login: %w", err)
	}

	provToken := &models.ProviderToken{
		UserID:       user.ID,
		ProviderID:   provider.ID,
		AccessToken:  models.EncryptedAtRest(accessToken),
		RefreshToken: models.EncryptedAtRest(refreshToken),
		Expiry:       expiry,
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
			if err := data.DeleteToken(db, data.ByUserID(user.ID)); err != nil && !errors.Is(err, internal.ErrNotFound) {
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
		return nil, "", fmt.Errorf("login user info: %w", err)
	}

	err = UpdateUserInfo(c, info, user, provider)
	if err != nil {
		return nil, "", fmt.Errorf("update info on login: %w", err)
	}

	sessionToken, err := data.IssueUserSessionToken(db, user, sessionDuration)
	if err != nil {
		return nil, "", fmt.Errorf("issue session token from code: %w", err)
	}

	user.LastSeenAt = time.Now()
	if err := data.UpdateUser(db, user, data.ByID(user.ID)); err != nil {
		return nil, "", fmt.Errorf("login update last seen: %w", err)
	}

	return user, sessionToken, nil
}

func RevokeToken(c *gin.Context) (*models.Token, error) {
	db, err := requireAuthorization(c, PermissionTokenRevoke)
	if err != nil {
		return nil, err
	}

	// added by the authentication middleware
	authentication, ok := c.MustGet("authentication").(string)
	if !ok {
		return nil, err
	}

	key := authentication[:models.TokenKeyLength]

	token, err := data.GetToken(db, data.ByKey(key))
	if err != nil {
		return nil, err
	}

	if err := data.DeleteToken(db, data.ByKey(key)); err != nil {
		return nil, err
	}

	return token, nil
}

// IssueAPIToken creates a configurable session token that can be used to directly interact with the Infra server API
func IssueAPIToken(c *gin.Context, apiToken *models.APIToken) (*models.Token, error) {
	db, err := requireAuthorization(c, PermissionAPITokenCreate)
	if err != nil {
		return nil, err
	}

	// do not let a caller create a token with more permissions than they have
	permissions, ok := c.MustGet("permissions").(string)
	if !ok {
		// there should have been permissions set by this point
		return nil, internal.ErrForbidden
	}

	if !AllRequired(strings.Split(permissions, " "), strings.Split(apiToken.Permissions, " ")) {
		return nil, fmt.Errorf("cannot create an API token with permission not granted to the token issuer")
	}

	if err := data.CreateAPIToken(db, apiToken); err != nil {
		return nil, fmt.Errorf("create api token: %w", err)
	}

	token := &models.Token{APITokenID: apiToken.ID, SessionDuration: apiToken.TTL}
	if err := data.CreateToken(db, token); err != nil {
		return nil, fmt.Errorf("create token: %w", err)
	}

	return token, nil
}

func ListAPITokens(c *gin.Context, name string) ([]models.APITokenTuple, error) {
	db, err := requireAuthorization(c, PermissionAPITokenRead)
	if err != nil {
		return nil, err
	}

	apiTokens, err := data.ListAPITokens(db, data.ByName(name))
	if err != nil {
		return nil, err
	}

	return apiTokens, nil
}

func RevokeAPIToken(c *gin.Context, id uid.ID) error {
	db, err := requireAuthorization(c, PermissionAPITokenDelete)
	if err != nil {
		return err
	}

	return data.DeleteAPIToken(db, id)
}

// IssueJWT creates a JWT that is presented to engines to assert authentication and claims
func IssueJWT(c *gin.Context, destination string) (string, *time.Time, error) {
	db, err := requireAuthorization(c, PermissionCredentialCreate)
	if err != nil {
		return "", nil, err
	}

	// added by the authentication middleware
	user, ok := c.MustGet("user").(*models.User)
	if !ok {
		return "", nil, errors.New("no jwt context user")
	}

	settings, err := data.GetSettings(db)
	if err != nil {
		return "", nil, err
	}

	return claims.CreateJWT(settings.PrivateJWK, user, destination)
}

// RetrieveUserProviderTokens gets the provider tokens that the current session token was created for
func RetrieveUserProviderTokens(c *gin.Context) (*models.ProviderToken, error) {
	db, err := requireAuthorization(c)
	if err != nil {
		return nil, err
	}

	// added by the authentication middleware
	user, ok := c.MustGet("user").(*models.User)
	if !ok {
		return nil, errors.New("no provider token context user")
	}

	return data.GetProviderToken(db, data.ByUserID(user.ID))
}

// UpdateProviderToken overwrites an existing set of provider tokens
func UpdateProviderToken(c *gin.Context, providerToken *models.ProviderToken) error {
	db, err := requireAuthorization(c)
	if err != nil {
		return err
	}

	return data.UpdateProviderToken(db, providerToken)
}
