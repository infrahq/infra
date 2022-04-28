package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
)

func TestSendAPIError(t *testing.T) {
	tests := []struct {
		err    error
		result api.Error
	}{
		{
			err: internal.ErrBadRequest,
			result: api.Error{
				Code:    400,
				Message: "bad request",
			},
		},
		{
			err: internal.ErrUnauthorized,
			result: api.Error{
				Code:    http.StatusUnauthorized,
				Message: "unauthorized",
			},
		},
		{
			err: internal.ErrForbidden,
			result: api.Error{
				Code:    http.StatusForbidden,
				Message: "forbidden",
			},
		},
		{
			err: internal.ErrDuplicate,
			result: api.Error{
				Code:    http.StatusConflict,
				Message: "duplicate record",
			},
		},
		{
			err: internal.ErrNotFound,
			result: api.Error{
				Code:    http.StatusNotFound,
				Message: "record not found",
			},
		},
		{
			err: internal.ErrNotImplemented,
			result: api.Error{
				Code:    http.StatusNotImplemented,
				Message: "not implemented",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.err.Error(), func(t *testing.T) {
			resp := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(resp)

			(&API{}).sendAPIError(c, test.err)

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
