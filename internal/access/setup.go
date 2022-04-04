package access

import (
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func SetupRequired(c *gin.Context) (bool, error) {
	// no authorization is setup yet
	db := getDB(c)

	settings, err := data.GetSettings(db)
	if err != nil {
		return false, err
	}

	return settings.SetupRequired, nil
}

func Setup(c *gin.Context) (string, *models.AccessKey, error) {
	// no authorization is setup yet
	db := getDB(c)

	settings, err := data.GetSettings(db)
	if err != nil {
		logging.S.Errorf("settings: %s", err)
		return "", nil, internal.ErrForbidden
	}

	if !settings.SetupRequired {
		return "", nil, internal.ErrForbidden
	}

	name := "admin"

	admin, err := data.GetIdentity(db, data.ByName(name))
	if err != nil {
		return "", nil, err
	}

	key := &models.AccessKey{
		Name:      fmt.Sprintf("%s-access-key", name),
		IssuedFor: admin.ID,
		ExpiresAt: time.Now().Add(math.MaxInt64).UTC(),
	}

	raw, err := data.CreateAccessKey(db, key)
	if err != nil {
		return "", nil, err
	}

	settings.SetupRequired = false
	if err := data.SaveSettings(db, settings); err != nil {
		return "", nil, err
	}

	return raw, key, nil
}
