package access

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
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

type CustomJWTClaims struct {
	Email       string `json:"email" validate:"required"`
	Destination string `json:"dest" validate:"required"`
	Nonce       string `json:"nonce" validate:"required"`
}

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
		User:            users[0],
		SessionDuration: sessionDuration,
	}

	if _, err := data.CreateToken(db, &token); err != nil {
		return nil, nil, err
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

var signatureAlgorithmFromKeyAlgorithm = map[string]string{
	"ED25519": "EdDSA", // elliptic curve 25519
}

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

	var sec jose.JSONWebKey
	if err := sec.UnmarshalJSON(settings.PrivateJWK); err != nil {
		return "", nil, err
	}

	algo, ok := signatureAlgorithmFromKeyAlgorithm[sec.Algorithm]
	if !ok {
		return "", nil, fmt.Errorf("unsupported algorithm")
	}

	options := &jose.SignerOptions{}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(algo), Key: sec}, options.WithType("JWT"))
	if err != nil {
		return "", nil, err
	}

	nonce, err := generate.CryptoRandom(10)
	if err != nil {
		return "", nil, err
	}

	now := time.Now()
	expiry := now.Add(time.Minute * 5)

	claim := jwt.Claims{
		Issuer:    "InfraHQ",
		NotBefore: jwt.NewNumericDate(now.Add(time.Minute * -5)), // adjust for clock drift
		Expiry:    jwt.NewNumericDate(expiry),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	custom := CustomJWTClaims{
		Email:       user.Email,
		Destination: destination,
		Nonce:       nonce,
	}

	raw, err := jwt.Signed(signer).Claims(claim).Claims(custom).CompactSerialize()
	if err != nil {
		return "", nil, err
	}

	return raw, &expiry, nil
}
