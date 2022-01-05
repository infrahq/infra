package claims

import (
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/models"
)

// Custom defines extended custom claims on top of the claims provided by go-jose
type Custom struct {
	Email       string   `json:"email" validate:"required"`
	Groups      []string `json:"groups"`
	Destination string   `json:"dest" validate:"required"`
	Nonce       string   `json:"nonce" validate:"required"`
}

var signatureAlgorithmFromKeyAlgorithm = map[string]string{
	"ED25519": "EdDSA", // elliptic curve 25519
}

// CreateJWT creates a JWT with the claims for the specified user and signs it
func CreateJWT(jwk []byte, user *models.User, destination string) (string, *time.Time, error) {
	// Warning: sec is a sensitive value, do not log it
	var sec jose.JSONWebKey
	if err := sec.UnmarshalJSON(jwk); err != nil {
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
	expiry := now.Add(time.Second * 30)

	claim := jwt.Claims{
		Issuer:    "InfraHQ",
		NotBefore: jwt.NewNumericDate(now.Add(time.Minute * -5)), // adjust for clock drift
		Expiry:    jwt.NewNumericDate(expiry),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	groupNames := make([]string, 0)
	for _, g := range user.Groups {
		groupNames = append(groupNames, g.Name)
	}

	custom := Custom{
		Email:       user.Email,
		Groups:      groupNames,
		Destination: destination,
		Nonce:       nonce,
	}

	raw, err := jwt.Signed(signer).Claims(claim).Claims(custom).CompactSerialize()
	if err != nil {
		return "", nil, err
	}

	return raw, &expiry, nil
}
