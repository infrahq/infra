package access

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
	db, err := requireAuthorization(c)
	if err != nil {
		return nil, nil, err
	}

	users, err := data.ListUsers(db, data.ByEmail(email))
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

	if err := data.CreateToken(db, &token); err != nil {
		return nil, nil, err
	}

	users[0].LastSeenAt = time.Now()

	if err := data.UpdateUser(db, &users[0], data.ByID(users[0].ID)); err != nil {
		return nil, nil, fmt.Errorf("user update fail: %w", err)
	}

	return &users[0], &token, nil
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

	apiTokens, err := data.ListAPITokens(db, &models.APIToken{Name: name})
	if err != nil {
		return nil, err
	}

	return apiTokens, nil
}

func RevokeAPIToken(c *gin.Context, id uuid.UUID) error {
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
