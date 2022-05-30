package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
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
			err:    fmt.Errorf("hide this: %w", internal.ErrUnauthorized),
			result: api.Error{Code: http.StatusUnauthorized, Message: "unauthorized"},
		},
		{
			err:    internal.ErrForbidden,
			result: api.Error{Code: http.StatusForbidden, Message: "forbidden"},
		},
		{
			err:    fmt.Errorf("hide this: %w", internal.ErrForbidden),
			result: api.Error{Code: http.StatusForbidden, Message: "forbidden"},
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
				Message: "value for name already exists in user",
			},
		},
		{
			err: validate.Struct(struct {
				Email string `validate:"required,email" json:"email"`
			}{}),
			result: api.Error{
				Code:    http.StatusBadRequest,
				Message: "Email: is required",
				FieldErrors: []api.FieldError{
					{
						FieldName: "Email",
						Errors:    []string{"is required"},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.err.Error(), func(t *testing.T) {
			resp := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(resp)

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
