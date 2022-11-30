package access

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slices"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
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

// SocialSignupDetails store the information about a sign-up from a successful OIDC authentication
type SocialSignupDetails struct {
	IDPAuth     *providers.IdentityProviderAuth
	RedirectURL string // stored on provider user to use refresh token in the future
	Provider    *models.Provider
}

// SocialSignup creates a user identity using a login from a social identity provider,
// and grants the identity "admin" access to Infra.
func SocialSignup(c *gin.Context, keyExpiresAt time.Time, baseDomain string, suDetails *SocialSignupDetails) (*NewOrgDetails, error) {
	rCtx := GetRequestContext(c)
	db := rCtx.DBTxn

	// start with automatically creating the org from the social login's domain
	org, err := createOrgFromEmail(db, suDetails.IDPAuth.Email, baseDomain)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}

	db = db.WithOrgID(org.ID)
	rCtx.DBTxn = db
	rCtx.Authenticated.Organization = org
	c.Set(RequestContextKey, rCtx)

	user := &models.ProviderUser{
		Email:        suDetails.IDPAuth.Email,
		RedirectURL:  suDetails.RedirectURL,
		AccessToken:  models.EncryptedAtRest(suDetails.IDPAuth.AccessToken),
		RefreshToken: models.EncryptedAtRest(suDetails.IDPAuth.RefreshToken),
		ExpiresAt:    suDetails.IDPAuth.AccessTokenExpiry,
		LastUpdate:   time.Now().UTC(),
		Active:       true,
	}

	identity, bearer, err := signupUser(c, keyExpiresAt, user)
	if err != nil {
		return nil, err
	}

	return &NewOrgDetails{
		Identity:     identity,
		Organization: org,
		Bearer:       bearer,
	}, nil
}

type OrgSignupDetails struct {
	Name      string
	Password  string
	Org       *models.Organization
	SubDomain string
}

// OrgSignup creates a user identity using the supplied name and password,
// generates an org name,
// and grants the identity "admin" access to Infra.
func OrgSignup(c *gin.Context, keyExpiresAt time.Time, baseDomain string, details OrgSignupDetails) (*NewOrgDetails, error) {
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

	user := &models.ProviderUser{
		ProviderID: data.InfraProvider(db).ID,
		Email:      details.Name,
		LastUpdate: time.Now().UTC(),
		Active:     true,
	}

	identity, bearer, err := signupUser(c, keyExpiresAt, user)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(details.Password), bcrypt.DefaultCost)
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

	return &NewOrgDetails{
		Identity:     identity,
		Organization: details.Org,
		Bearer:       bearer,
	}, nil
}

func createOrgFromEmail(tx data.WriteTxn, emailAddr, infraDomain string) (*models.Organization, error) {
	orgName, err := getOrgDomainFromEmail(emailAddr)
	if err != nil {
		return nil, fmt.Errorf("get email domain: %w", err)
	}

	domain, err := sanitizeOrgDomainFromName(tx, orgName, infraDomain)
	if err != nil {
		return nil, fmt.Errorf("unable to sanitize org domain from name: %w", err)
	}
	org := &models.Organization{
		Name:   orgName,
		Domain: domain,
	}

	if err := data.CreateOrganization(tx, org); err != nil {
		return nil, fmt.Errorf("could not create unique org: %w", err)
	}

	return org, nil
}

// sanitizeOrgDomainFromName attempts to create a unique valid org domain from the name of an org
func sanitizeOrgDomainFromName(tx data.ReadTxn, name string, infraDomain string) (string, error) {
	if len(name) == 0 {
		// this should not be possible
		return "", fmt.Errorf("empty org name")
	}
	sub := name
	if len(sub) > 63 {
		sub = sub[:63]
	}

	// is the length of the length subdomain less than our minimum length (4) or a reserved domain?
	needsPostfix := len(name) < 4 || slices.Contains(api.ReservedSubdomains, sub)
	// does another org already use this subdomain?
	if _, err := data.GetOrganization(tx, data.GetOrganizationOptions{ByDomain: SanitizedDomain(sub, infraDomain)}); !errors.Is(err, internal.ErrNotFound) {
		logging.L.Debug().Err(err).Msg("failed to automatically create unique org from social sign-up, this may be expected")
		needsPostfix = true
	}
	if needsPostfix {
		postfix := generate.MathRandom(3, generate.CharsetNumbers) // get 3 random numbers to append to the name
		sub = fmt.Sprintf("%s-%s", sub, postfix)
	}
	return SanitizedDomain(sub, infraDomain), nil
}

func getOrgDomainFromEmail(emailAddr string) (string, error) {
	// get the domain after the '@' in the email
	domain, err := email.Domain(emailAddr)
	if err != nil {
		return "", err
	}
	domainParts := strings.Split(domain, ".")
	if len(domainParts) <= 1 {
		// should not happen
		return "", fmt.Errorf("invalid email domain")
	}
	baseDomain := domainParts[len(domainParts)-2] // the domain before the TLD

	// Do not create the org domain for a generic email address not tied to an org
	// TODO: add more domain checks here as more social sign-up is possible
	if baseDomain == "gmail" || baseDomain == "googlemail" {
		// set the org name from the email identifier
		addr := strings.Split(emailAddr, "@")
		baseDomain = addr[0]
	}

	// a length of 254 is chosen as RFC3696 Errata ID 1690 states that the total length of an email address must not exceed 254 characters
	if len(baseDomain) > 254 {
		baseDomain = baseDomain[:254]
	}

	return baseDomain, nil
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
