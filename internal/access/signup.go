package access

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func SignupEnabled(c *gin.Context) (bool, error) {
	// no authorization is setup yet
	db := getDB(c)

	settings, err := data.GetSettings(db)
	if err != nil {
		return false, err
	}

	return settings.SignupEnabled, nil
}

func Signup(c *gin.Context, name, password string) (*models.Identity, error) {
	// no authorization is setup yet
	db := getDB(c)

	settings, err := data.GetSettings(db)
	if err != nil {
		logging.S.Errorf("settings: %s", err)
		return nil, internal.ErrForbidden
	}

	if !settings.SignupEnabled {
		return nil, internal.ErrForbidden
	}

	identity := &models.Identity{
		Name: name,
		Kind: models.UserKind,
	}

	if err := data.CreateIdentity(db, identity); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	credential := &models.Credential{
		IdentityID:   identity.ID,
		PasswordHash: hash,
	}

	if err := data.CreateCredential(db, credential); err != nil {
		return nil, err
	}

	grant := &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(identity.ID),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
	}

	if err := data.CreateGrant(db, grant); err != nil {
		return nil, err
	}

	settings.SignupEnabled = false
	if err := data.SaveSettings(db, settings); err != nil {
		return nil, err
	}

	return identity, nil
}
