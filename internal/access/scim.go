package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListProviderUsers(c *gin.Context, providerID uid.ID, p *data.SCIMParameters) ([]models.ProviderUser, error) {
	db, err := RequireInfraRole(c, models.InfraSCIMRole)
	if err != nil {
		return nil, HandleAuthErr(err, "scim", "list users", models.InfraSCIMRole)
	}

	users, err := data.ListProviderUsers(db, providerID, p)
	if err != nil {
		return nil, fmt.Errorf("list provider users: %w", err)
	}
	return users, nil
}
