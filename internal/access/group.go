package access

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListGroups(c *gin.Context, name string, providerID uid.ID) ([]models.Group, error) {
	db, err := requireInfraRole(c, AdminRole, ViewRole, ConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListGroups(db, data.ByName(name), data.ByProviderID(providerID))
}

func CreateGroup(c *gin.Context, group *models.Group) error {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return err
	}

	return data.CreateGroup(db, group)
}

func GetGroup(c *gin.Context, id uid.ID) (*models.Group, error) {
	user := CurrentUser(c)
	userGroups := make(map[uid.ID]bool)

	var db *gorm.DB

	if user != nil {
		lookupDB := getDB(c)
		groups, err := data.ListUserGroups(lookupDB, user.ID)
		if err != nil {
			return nil, err
		}

		for _, g := range groups {
			userGroups[g.ID] = true
		}
	}

	if userGroups[id] {
		// user is in group
		db = getDB(c)
	} else {
		var err error
		db, err = requireInfraRole(c, AdminRole, ViewRole, ConnectorRole)
		if err != nil {
			return nil, err
		}
	}

	return data.GetGroup(db, data.ByID(id))
}

func ListUserGroups(c *gin.Context, userID uid.ID) ([]models.Group, error) {
	user := CurrentUser(c)

	var db *gorm.DB
	if user != nil && user.ID == userID {
		db = getDB(c)
	} else {
		var err error
		db, err = requireInfraRole(c, AdminRole, ViewRole)
		if err != nil {
			return nil, err
		}
	}

	return data.ListUserGroups(db, userID)
}
