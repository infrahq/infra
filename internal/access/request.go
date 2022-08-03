package access

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
)

const RequestContextKey = "requestContext"

// RequestContext stores the http.Request, and values derived from the request
// like the authenticated user. It also provides a database transaction.
type RequestContext struct {
	Request       *http.Request
	DBTxn         *gorm.DB
	Authenticated Authenticated
}

// Authenticated stores data about the authenticated user. If the AccessKey or
// User are nil, it indicates that no user was authenticated.
type Authenticated struct {
	AccessKey *models.AccessKey
	User      *models.Identity
}
