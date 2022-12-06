package access

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

// NewOrgDetails are details about the identity, org, and access key after a sign-up is successful
type NewOrgDetails struct {
	Identity     *models.Identity
	Organization *models.Organization
	Bearer       string
}

// SocialSignup stores the information about a sign-up from a successful OIDC authentication
type SignupSocial struct {
	IDPAuth     *providers.IdentityProviderAuth
	RedirectURL string // stored on provider user to use refresh token in the future
	Provider    *models.Provider
}

// SocialUser allows a user to sign-up with an email and a password
type SignupUser struct {
	Name     string
	Password string
}

type SignupDetails struct {
	User      *SignupUser
	Social    *SignupSocial
	Org       *models.Organization
	SubDomain string
}

// Signup creates an organization, and grants an identity "admin" access to Infra.
func Signup(c *gin.Context, keyExpiresAt time.Time, baseDomain string, details *SignupDetails) (*NewOrgDetails, error) {
	if details.Social == nil && details.User == nil {
		return nil, fmt.Errorf("sign-up requires social login details or user details")
	}
	rCtx := GetRequestContext(c)
	db := rCtx.DBTxn

	details.Org.Domain = SanitizedDomain(details.SubDomain, baseDomain)

	if err := data.CreateOrganization(db, details.Org); err != nil {
		return nil, fmt.Errorf("create org on sign-up: %w", err)
	}

	db = db.WithOrgID(details.Org.ID)
	rCtx.DBTxn = db
	rCtx.Authenticated.Organization = details.Org
	c.Set(RequestContextKey, rCtx)

	var identity *models.Identity
	bearer := ""
	switch {
	case details.User != nil:
		// username/password sign-up
		user := &models.ProviderUser{
			ProviderID: data.InfraProvider(db).ID,
			Email:      details.User.Name,
			LastUpdate: time.Now().UTC(),
			Active:     true,
		}

		var err error
		identity, bearer, err = signupUser(c, keyExpiresAt, user)
		if err != nil {
			return nil, err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(details.User.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password on sign-up: %w", err)
		}

		credential := &models.Credential{
			IdentityID:   identity.ID,
			PasswordHash: hash,
		}

		if err := data.CreateCredential(db, credential); err != nil {
			return nil, fmt.Errorf("create credential on sign-up: %w", err)
		}
	case details.Social != nil:
		// sign-up with social (google)
		user := &models.ProviderUser{
			Email:        details.Social.IDPAuth.Email,
			RedirectURL:  details.Social.RedirectURL,
			AccessToken:  models.EncryptedAtRest(details.Social.IDPAuth.AccessToken),
			RefreshToken: models.EncryptedAtRest(details.Social.IDPAuth.RefreshToken),
			ExpiresAt:    details.Social.IDPAuth.AccessTokenExpiry,
			LastUpdate:   time.Now().UTC(),
			Active:       true,
		}

		var err error
		identity, bearer, err = signupUser(c, keyExpiresAt, user)
		if err != nil {
			return nil, err
		}
	default:
		// this should have been caught by the initial error check
		return nil, fmt.Errorf("sign-up requires social login or user credentials")
	}

	return &NewOrgDetails{
		Identity:     identity,
		Organization: details.Org,
		Bearer:       bearer,
	}, nil
}

// signupUser creates the user identity and grants for a new org
func signupUser(c *gin.Context, keyExpiresAt time.Time, user *models.ProviderUser) (*models.Identity, string, error) {
	rCtx := GetRequestContext(c)
	tx := rCtx.DBTxn

	identity := &models.Identity{
		Name: user.Email,
	}
	if err := data.CreateIdentity(tx, identity); err != nil {
		return nil, "", fmt.Errorf("create identity on sign-up: %w", err)
	}
	user.IdentityID = identity.ID

	// create the provider user with the specified fields
	err := data.ProvisionProviderUser(tx, user)
	if err != nil {
		return nil, "", fmt.Errorf("create provider user on sign-up: %w", err)
	}

	err = data.CreateGrant(tx, &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(identity.ID),
		Privilege: models.InfraAdminRole,
		Resource:  ResourceInfraAPI,
		CreatedBy: identity.ID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create grant on sign-up: %w", err)
	}

	// grant the user a session on initial sign-up
	accessKey := &models.AccessKey{
		IssuedFor:     identity.ID,
		IssuedForName: identity.Name,
		ProviderID:    user.ProviderID,
		ExpiresAt:     keyExpiresAt,
		Scopes:        []string{models.ScopeAllowCreateAccessKey},
	}

	bearer, err := data.CreateAccessKey(tx, accessKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create access key after sign-up: %w", err)
	}

	// Update the request context so that logging middleware can include the userID
	rCtx.Authenticated.User = identity
	c.Set(RequestContextKey, rCtx)

	return identity, bearer, nil
}
