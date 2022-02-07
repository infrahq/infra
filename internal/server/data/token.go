package data

import (
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

var signatureAlgorithmFromKeyAlgorithm = map[string]string{
	"ED25519": "EdDSA", // elliptic curve 25519
}

func createJWT(db *gorm.DB, email string, groups []string, expires time.Time) (string, error) {
	settings, err := GetSettings(db)
	if err != nil {
		return "", err
	}

	var sec jose.JSONWebKey
	if err := sec.UnmarshalJSON(settings.PrivateJWK); err != nil {
		return "", err
	}

	algo, ok := signatureAlgorithmFromKeyAlgorithm[sec.Algorithm]
	if !ok {
		return "", fmt.Errorf("unsupported algorithm")
	}

	options := &jose.SignerOptions{}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(algo), Key: sec}, options.WithType("JWT"))
	if err != nil {
		return "", err
	}

	nonce, err := generate.CryptoRandom(10)
	if err != nil {
		return "", err
	}

	now := time.Now()

	claim := jwt.Claims{
		NotBefore: jwt.NewNumericDate(now.Add(time.Minute * -5)), // adjust for clock drift
		Expiry:    jwt.NewNumericDate(expires),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	custom := claims.Custom{
		Email:  email,
		Groups: groups,
		Nonce:  nonce,
	}

	raw, err := jwt.Signed(signer).Claims(claim).Claims(custom).CompactSerialize()
	if err != nil {
		return "", err
	}

	return raw, nil
}

func CreateToken(db *gorm.DB, userID uid.ID) (token *models.Token, err error) {
	user, err := GetUser(db, ByID(userID))
	if err != nil {
		return nil, err
	}

	userGroups, err := ListUserGroups(db, userID)
	if err != nil {
		return nil, err
	}

	var groups []string
	for _, g := range userGroups {
		groups = append(groups, g.Name)
	}

	expires := time.Now().Add(time.Minute * 5)

	jwt, err := createJWT(db, user.Email, groups, expires)
	if err != nil {
		return nil, err
	}

	return &models.Token{Token: jwt, Expires: expires}, nil
}
