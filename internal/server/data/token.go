package data

import (
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

var signatureAlgorithmFromKeyAlgorithm = map[string]string{
	"ED25519": "EdDSA", // elliptic curve 25519
}

func createJWT(db ReadTxn, identity *models.Identity, groups []string, expires time.Time) (string, error) {
	settings, err := GetSettings(db)
	if err != nil {
		return "", err
	}

	var sec jose.JSONWebKey
	if err := sec.UnmarshalJSON([]byte(settings.PrivateJWK)); err != nil {
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

	now := time.Now().UTC()

	claim := jwt.Claims{
		NotBefore: jwt.NewNumericDate(now.Add(time.Minute * -5)), // adjust for clock drift
		Expiry:    jwt.NewNumericDate(expires),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
	}

	custom := claims.Custom{
		Name:   identity.Name,
		Groups: groups,
	}

	raw, err := jwt.Signed(signer).Claims(claim).Claims(custom).CompactSerialize()
	if err != nil {
		return "", err
	}

	return raw, nil
}

func CreateIdentityToken(db ReadTxn, identityID uid.ID) (token *models.Token, err error) {
	identity, err := GetIdentity(db, GetIdentityOptions{ByID: identityID})
	if err != nil {
		return nil, err
	}

	identityGroups, err := ListGroups(db, ListGroupsOptions{ByGroupMember: identityID})
	if err != nil {
		return nil, err
	}

	var groups []string
	for _, g := range identityGroups {
		groups = append(groups, g.Name)
	}

	expires := time.Now().Add(time.Minute * 5).UTC()

	jwt, err := createJWT(db, identity, groups, expires)
	if err != nil {
		return nil, err
	}

	return &models.Token{Token: jwt, Expires: expires}, nil
}
