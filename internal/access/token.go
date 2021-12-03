package access

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

const (
	PermissionToken       Permission = "infra.token.*" // nolint:gosec
	PermissionTokenRead   Permission = "infra.token.read"
	PermissionTokenRevoke Permission = "infra.token.revoke" // nolint:gosec

<<<<<<< HEAD
	PermissionAPIKey       Permission = "infra.apiKey.*"      // nolint:gosec
	PermissionAPIKeyIssue  Permission = "infra.apiKey.issue"  // nolint:gosec
	PermissionAPIKeyRead   Permission = "infra.apiKey.read"   // nolint:gosec
	PermissionAPIKeyRevoke Permission = "infra.apiKey.revoke" // nolint:gosec
||||||| parent of 9f521ce (Rename API keys to API tokens)
	PermissionAPIKey       Permission = "infra.apiKey.*"      // nolint:gosec
	PermissionAPIKeyIssue  Permission = "infra.apiKey.issue"  // nolint:gosec
	PermissionAPIKeyList   Permission = "infra.apiKey.list"   // nolint:gosec
	PermissionAPIKeyRevoke Permission = "infra.apiKey.revoke" // nolint:gosec
=======
	PermissionAPIToken       Permission = "infra.apiToken.*"      // nolint:gosec
	PermissionAPITokenIssue  Permission = "infra.apiToken.issue"  // nolint:gosec
	PermissionAPITokenList   Permission = "infra.apiToken.list"   // nolint:gosec
	PermissionAPITokenRevoke Permission = "infra.apiToken.revoke" // nolint:gosec
>>>>>>> 9f521ce (Rename API keys to API tokens)

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

func IssueAPIToken(c *gin.Context, apiToken *models.APIToken) (*models.APIToken, error) {
	db, err := RequireAuthorization(c, PermissionAPITokenIssue)
	if err != nil {
		return nil, err
	}

	return data.CreateAPIToken(db, apiToken)
}

func ListAPITokens(c *gin.Context, name string) ([]models.APIToken, error) {
	db, err := RequireAuthorization(c, PermissionAPITokenList)
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
	db, err := RequireAuthorization(c, PermissionAPITokenRevoke)
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
