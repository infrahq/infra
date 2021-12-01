package access

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/generate"
)

const (
	PermissionToken       Permission = "infra.token.*" // nolint:gosec
	PermissionTokenRead   Permission = "infra.token.read"
	PermissionTokenRevoke Permission = "infra.token.revoke" // nolint:gosec

	PermissionAPIKey       Permission = "infra.apiKey.*"      // nolint:gosec
	PermissionAPIKeyIssue  Permission = "infra.apiKey.issue"  // nolint:gosec
	PermissionAPIKeyList   Permission = "infra.apiKey.list"   // nolint:gosec
	PermissionAPIKeyRevoke Permission = "infra.apiKey.revoke" // nolint:gosec

	PermissionCredentialCreate Permission = "infra.credential.create" //nolint:gosec
)

type CustomJWTClaims struct {
	Email       string `json:"email" validate:"required"`
	Destination string `json:"dest" validate:"required"`
	Nonce       string `json:"nonce" validate:"required"`
}

func IssueToken(c *gin.Context, email string, sessionDuration time.Duration) (*data.User, *data.Token, error) {
	db, _, err := RequireAuthorization(c, Permission(""))
	if err != nil {
		return nil, nil, err
	}

	users, err := data.ListUsers(db, &data.User{Email: email})
	if err != nil {
		return nil, nil, err
	}

	if len(users) != 1 {
		return nil, nil, fmt.Errorf("unknown user")
	}

	permissions := strings.Join([]string{
		string(PermissionUserRead),
		string(PermissionTokenRevoke),
		string(PermissionCredentialCreate),
	}, " ")

	token := data.Token{
		User:            users[0],
		SessionDuration: sessionDuration,
		Permissions:     permissions,
	}

	if _, err := data.CreateToken(db, &token); err != nil {
		return nil, nil, err
	}

	return &users[0], &token, nil
}

func RevokeToken(c *gin.Context) (*data.Token, error) {
	db, authorization, err := RequireAuthorization(c, PermissionTokenRevoke)
	if err != nil {
		return nil, err
	}

	key := authorization[:data.TokenKeyLength]

	token, err := data.GetToken(db, &data.Token{Key: key})
	if err != nil {
		return nil, err
	}

	if err := data.DeleteToken(db, &data.Token{Key: key}); err != nil {
		return nil, err
	}

	return token, nil
}

func IssueAPIKey(c *gin.Context, template *api.InfraAPIKeyCreateRequest) (*data.APIKey, error) {
	db, _, err := RequireAuthorization(c, PermissionAPIKeyIssue)
	if err != nil {
		return nil, err
	}

	var apiKey data.APIKey
	if err := apiKey.FromAPICreateRequest(template); err != nil {
		return nil, err
	}

	return data.CreateAPIKey(db, &apiKey)
}

func ListAPIKeys(c *gin.Context, name string) ([]data.APIKey, error) {
	db, _, err := RequireAuthorization(c, PermissionAPIKeyList)
	if err != nil {
		return nil, err
	}

	apiKeys, err := data.ListAPIKeys(db, &data.APIKey{Name: name})
	if err != nil {
		return nil, err
	}

	return apiKeys, nil
}

func RevokeAPIKey(c *gin.Context, id string) error {
	db, _, err := RequireAuthorization(c, PermissionAPIKeyRevoke)
	if err != nil {
		return err
	}

	token, err := data.NewAPIKey(id)
	if err != nil {
		return err
	}

	return data.DeleteAPIKey(db, token)
}

var signatureAlgorithmFromKeyAlgorithm = map[string]string{
	"ED25519": "EdDSA", // elliptic curve 25519
}

func IssueJWT(c *gin.Context, destination string) (string, *time.Time, error) {
	db, authorization, err := RequireAuthorization(c, PermissionCredentialCreate)
	if err != nil {
		return "", nil, err
	}

	user, err := data.GetUser(db, data.UserTokenSelector(db, authorization))
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
