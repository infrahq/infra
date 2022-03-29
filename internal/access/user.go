package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isUserSelf is used by authorization checks to see if the calling user is requesting their own attributes
func isUserSelf(c *gin.Context, requestedResourceID uid.ID) (bool, error) {
	user := CurrentUser(c)
	return user != nil && user.ID == requestedResourceID, nil
}

func CurrentUser(c *gin.Context) *models.User {
	userObj, exists := c.Get("user")
	if !exists {
		return nil
	}

	user, ok := userObj.(*models.User)
	if !ok {
		return nil
	}

	return user
}

func GetUser(c *gin.Context, id uid.ID) (*models.User, error) {
	db, err := hasAuthorization(c, id, isUserSelf, models.InfraAdminRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.GetUser(db, data.ByID(id))
}

func CreateUser(c *gin.Context, user *models.User) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.CreateUser(db, user)
}

func DeleteUser(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	// if a user does not exist in the Infra provider this won't be found, but we can proceed
	credential, err := data.GetCredential(db, data.ByIdentity(uid.NewUserPolymorphicID(id)))
	if err != nil && !errors.Is(err, internal.ErrNotFound) {
		return fmt.Errorf("get delete user creds: %w", err)
	}

	if credential != nil {
		err := data.DeleteCredential(db, credential.ID)
		if err != nil {
			return fmt.Errorf("delete user creds: %w", err)
		}
	}

	return data.DeleteUser(db, id)
}

func ListUsers(c *gin.Context, email string, providerID uid.ID) ([]models.User, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListUsers(db, data.ByEmail(email), data.ByProviderID(providerID))
}

func UpdateUserInfo(c *gin.Context, info *authn.UserInfo, user *models.User, provider *models.Provider) error {
	// no auth, this is not publically exposed
	db := getDB(c)

	// add user to groups they are currently in
	var groups []models.Group

	if info.Groups != nil {
		for i := range *info.Groups {
			name := (*info.Groups)[i]

			group, err := data.GetGroup(db, data.ByName(name))
			if err != nil {
				if !errors.Is(err, internal.ErrNotFound) {
					return fmt.Errorf("get group: %w", err)
				}

				group = &models.Group{Name: name}

				if err = data.CreateGroup(db, group); err != nil {
					return fmt.Errorf("create group: %w", err)
				}
			}

			groups = append(groups, *group)
		}
	}

	// remove user from groups they are no longer in
	return data.BindUserGroups(db, user, groups...)
}
