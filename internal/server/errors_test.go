package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/validate"
)

func TestSendAPIError(t *testing.T) {
	tests := []struct {
		err    error
		result api.Error
	}{
		{
			err:    internal.ErrBadRequest,
			result: api.Error{Code: http.StatusBadRequest, Message: "bad request"},
		},
		{
			err: fmt.Errorf("not right: %w", internal.ErrBadRequest),
			result: api.Error{
				Code:    http.StatusBadRequest,
				Message: "not right: bad request",
			},
		},
		{
			err:    internal.ErrUnauthorized,
			result: api.Error{Code: http.StatusUnauthorized, Message: "unauthorized"},
		},
		{
			err: validate.Error{"fieldname": []string{"is required"}},
			result: api.Error{
				Code:    http.StatusBadRequest,
				Message: "validation failed: fieldname: is required",
				FieldErrors: []api.FieldError{
					{FieldName: "fieldname", Errors: []string{"is required"}},
				},
			},
		},
		{
			err:    fmt.Errorf("hide this: %w", internal.ErrUnauthorized),
			result: api.Error{Code: http.StatusUnauthorized, Message: "unauthorized"},
		},
		{
			err:    data.ErrAccessKeyExpired,
			result: api.Error{Code: http.StatusUnauthorized, Message: "unauthorized: " + data.ErrAccessKeyExpired.Error()},
		},
		{
			err:    data.ErrAccessKeyDeadlineExceeded,
			result: api.Error{Code: http.StatusUnauthorized, Message: "unauthorized: " + data.ErrAccessKeyDeadlineExceeded.Error()},
		},
		{
			err: access.AuthorizationError{
				Resource:      "provider",
				Operation:     "create",
				RequiredRoles: []string{"admin"},
			},
			result: api.Error{
				Code:    http.StatusForbidden,
				Message: "you do not have permission to create provider, requires role admin",
			},
		},
		{
			err:    internal.ErrNotFound,
			result: api.Error{Code: http.StatusNotFound, Message: "record not found"},
		},
		{
			err:    internal.ErrNotImplemented,
			result: api.Error{Code: http.StatusNotImplemented, Message: "not implemented"},
		},
		{
			err: data.UniqueConstraintError{Table: "user", Column: "name"},
			result: api.Error{
				Code:    http.StatusConflict,
				Message: "a user with that name already exists",
				FieldErrors: []api.FieldError{
					{FieldName: "name", Errors: []string{"a user with that name already exists"}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.err.Error(), func(t *testing.T) {
			resp := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(resp)
			c.Request = &http.Request{
				Method:     http.MethodPost,
				URL:        &url.URL{Path: "/api/path"},
				RemoteAddr: "10.10.10.10:34124",
			}

			sendAPIError(c, test.err)

			assert.Equal(t, test.result.Code, int32(resp.Result().StatusCode))
			actual := &api.Error{}
			err := json.NewDecoder(resp.Body).Decode(actual)
			assert.NilError(t, err)

			assert.Equal(t, test.result.Code, actual.Code)
			assert.Equal(t, test.result.Message, actual.Message)

			assert.DeepEqual(t, test.result.FieldErrors, actual.FieldErrors)
		})
	}
}
