package access

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
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

// IssueUserToken creates a session token that a user presents to the Infra server for authentication
func IssueUserToken(c *gin.Context, email string, sessionDuration time.Duration) (*models.User, *models.Token, error) {
	db, err := RequireAuthorization(c, Permission(""))
	if err != nil {
		return nil, nil, err
	}

	users, err := data.ListUsers(db, &models.User{Email: email})
	if err != nil {
		return nil, nil, err
	}

	if len(users) != 1 {
		return nil, nil, fmt.Errorf("unknown user")
	}

	token := models.Token{
		UserID:          users[0].ID,
		SessionDuration: sessionDuration,
	}

	if _, err := data.CreateToken(db, &token); err != nil {
		return nil, nil, err
	}

	users[0].LastSeenAt = time.Now()

	if err := data.UpdateUser(db, &users[0], data.ByUUID(users[0].ID)); err != nil {
		return nil, nil, fmt.Errorf("user update fail: %w", err)
	}

	return &users[0], &token, nil
}

func RevokeToken(c *gin.Context) (*models.Token, error) {
	db, err := RequireAuthorization(c, PermissionTokenRevoke)
	if err != nil {
		return nil, err
	}

	// added by the authentication middleware
	authentication, ok := c.MustGet("authentication").(string)
	if !ok {
		return nil, err
	}

	key := authentication[:models.TokenKeyLength]

	token, err := data.GetToken(db, &models.Token{Key: key})
	if err != nil {
		return nil, err
	}

	if err := data.DeleteToken(db, &models.Token{Key: key}); err != nil {
		return nil, err
	}

	return token, nil
}

// IssueAPIToken creates a configurable session token that can be used to directly interact with the Infra server API
func IssueAPIToken(c *gin.Context, apiToken *models.APIToken) (*models.APIToken, *models.Token, error) {
	db, err := RequireAuthorization(c, PermissionAPITokenCreate)
	if err != nil {
		return nil, nil, err
	}

	// do not let a caller create a token with more permissions than they have
	permissions, ok := c.MustGet("permissions").(string)
	if !ok {
		// there should have been permissions set by this point
		return nil, nil, internal.ErrForbidden
	}

	if !AllRequired(strings.Split(permissions, " "), strings.Split(apiToken.Permissions, " ")) {
		return nil, nil, fmt.Errorf("cannot create an API token with permission not granted to the token issuer")
	}

	return data.CreateAPIToken(db, apiToken, &models.Token{})
}

func ListAPITokens(c *gin.Context, name string) ([]models.APITokenTuple, error) {
	db, err := RequireAuthorization(c, PermissionAPITokenRead)
	if err != nil {
		return nil, err
	}

	apiTokens, err := data.ListAPITokens(db, &models.APIToken{Name: name})
	if err != nil {
		return nil, err
	}

	return apiTokens, nil
}

func RevokeAPIToken(c *gin.Context, id string) error {
	db, err := RequireAuthorization(c, PermissionAPITokenDelete)
	if err != nil {
		return err
	}

	token, err := models.NewAPIToken(id)
	if err != nil {
		return err
	}

	return data.DeleteAPIToken(db, token)
}

// IssueJWT creates a JWT that is presented to engines to assert authentication and claims
func IssueJWT(c *gin.Context, destination string) (string, *time.Time, error) {
	db, err := RequireAuthorization(c, PermissionCredentialCreate)
	if err != nil {
		return "", nil, err
	}

	// added by the authentication middleware
	authentication, ok := c.MustGet("authentication").(string)
	if !ok {
		return "", nil, err
	}

	user, err := data.GetUser(db, data.UserTokenSelector(db, authentication))
	if err != nil {
		return "", nil, err
	}

	settings, err := data.GetSettings(db)
	if err != nil {
		return "", nil, err
	}

	return claims.CreateJWT(settings.PrivateJWK, user, destination)
}
