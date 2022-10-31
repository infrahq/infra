package access

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
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
	Response *ResponseMetadata
}

// Authenticated stores data about the authenticated user. If the AccessKey or
// User are nil, it indicates that no user was authenticated.
type Authenticated struct {
	AccessKey    *models.AccessKey
	User         *models.Identity
	Organization *models.Organization
}

// ResponseMetadata is accumulated by API endpoints and used for logging and
// reporting.
type ResponseMetadata struct {
	// logFields is a slice of function that can add fields to the API
	// request log entry.
	logFields []func(event *zerolog.Event)
}

func (r *ResponseMetadata) AddLogFields(fn func(event *zerolog.Event)) {
	if r == nil {
		return
	}
	r.logFields = append(r.logFields, fn)
}

func (r *ResponseMetadata) ApplyLogFields(event *zerolog.Event) {
	if r == nil {
		return
	}
	for _, fn := range r.logFields {
		fn(event)
	}
}
