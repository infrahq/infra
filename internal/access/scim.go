package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func ListProviderUsers(c *gin.Context, p *data.SCIMParameters) ([]models.ProviderUser, error) {
	ctx := GetRequestContext(c)
	// IssuedFor will match no providers if called with a regular access key. When called with
	// a SCIM access key it will be the provider ID. This effectively restricts this endpoint to
	// only SCIM access keys.
	opts := data.ListProviderUsersOptions{
		ByProviderID:   ctx.Authenticated.AccessKey.IssuedFor,
		SCIMParameters: p,
	}
	users, err := data.ListProviderUsers(ctx.DBTxn, opts)
	if err != nil {
		return nil, fmt.Errorf("list provider users: %w", err)
	}
	return users, nil
}
