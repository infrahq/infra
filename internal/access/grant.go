package access

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := requireInfraRole(c, AdminRole, ViewRole)
	if err != nil {
		return nil, err
	}

	return data.GetGrant(db, data.ByID(id), data.NotCreatedBySystem())
}

func ListGrants(c *gin.Context, identity uid.PolymorphicID, resource string, privilege string) ([]models.Grant, error) {
	db, err := requireInfraRole(c, AdminRole, ViewRole, ConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListGrants(db, data.ByIdentity(identity), data.ByResource(resource), data.ByPrivilege(privilege), data.NotCreatedBySystem())
}

func ListUserGrants(c *gin.Context, userID uid.ID) ([]models.Grant, error) {
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

	return data.ListUserGrants(db, userID)
}

func ListMachineGrants(c *gin.Context, machineID uid.ID) ([]models.Grant, error) {
	machine := CurrentMachine(c)

	var db *gorm.DB
	if machine != nil && machine.ID == machineID {
		db = getDB(c)
	} else {
		var err error
		db, err = requireInfraRole(c, AdminRole, ViewRole)
		if err != nil {
			return nil, err
		}
	}

	return data.ListMachineGrants(db, machineID)
}

func ListGroupGrants(c *gin.Context, groupID uid.ID) ([]models.Grant, error) {
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

	if userGroups[groupID] {
		// user is in group
		db = getDB(c)
	} else {
		var err error
		db, err = requireInfraRole(c, AdminRole, ViewRole)
		if err != nil {
			return nil, err
		}
	}

	return data.ListGroupGrants(db, groupID)
}

func CreateGrant(c *gin.Context, grant *models.Grant) error {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return err
	}

	return data.CreateGrant(db, grant)
}

func DeleteGrant(c *gin.Context, id uid.ID) error {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return err
	}

	return data.DeleteGrants(db, data.ByID(id), data.NotCreatedBySystem())
}
