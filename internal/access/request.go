package access

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

const RequestContextKey = "requestContext"

// RequestContext stores the http.Request, and values derived from the request
// like the authenticated user. It also provides a database transaction.
type RequestContext struct {
	Request       *http.Request
	DBTxn         *data.Transaction
	Authenticated Authenticated

	// DataDB is the full database connection pool that can be used to
	// start transactions. Most routes should use DBTxn and should not use
	// DataDB directly.
	DataDB *data.DB

	// Response is a mutable field. It can be modified by API handlers to add
	// new response metadata.
	Response *Response
}

// Authenticated stores data about the authenticated user. If the AccessKey or
// User are nil, it indicates that no user was authenticated.
type Authenticated struct {
	AccessKey    *models.AccessKey
	User         *models.Identity
	Organization *models.Organization
}

// Response is accumulated by API endpoints and used for logging and
// reporting.
type Response struct {
	// HTTPWriter is the http.ResponseWriter that will be used to write the response.
	// In most cases the HTTPWriter should only be used to write response headers
	// or cookies using Header().
	// It is only safe to call Write and WriteHeader if the API handler returns
	// an empty response and no error.
	HTTPWriter http.ResponseWriter

	// LoginUserID stores the user ID for login, and signup type endpoints so that
	// the ID can be included in the API request log entry.
	LoginUserID uid.ID

	// SignupOrgID stores the organization ID for a new signup so that the ID
	// can be included in the API request log entry.
	SignupOrgID uid.ID

	// logFields is a slice of function that can add fields to the API
	// request log entry.
	logFields []func(event *zerolog.Event)
}

func (r *Response) AddLogFields(fn func(event *zerolog.Event)) {
	if r == nil {
		return
	}
	r.logFields = append(r.logFields, fn)
}

func (r *Response) ApplyLogFields(event *zerolog.Event) {
	if r == nil {
		return
	}
	for _, fn := range r.logFields {
		fn(event)
	}
}
