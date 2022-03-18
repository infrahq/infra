package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
)

func TestSendAPIError(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

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
		{
			err: validate.Struct(struct {
				Email string `validate:"required,email" json:"email"`
			}{Email: "foo#example!com"}),
			result: api.Error{
				Code:    http.StatusBadRequest,
				Message: `Email: failed the "email" check`,
				FieldErrors: []api.FieldError{
					{
						FieldName: "Email",
						Errors:    []string{`failed the "email" check`},
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

			require.EqualValues(t, test.result.Code, resp.Result().StatusCode)
			actual := &api.Error{}
			err := json.NewDecoder(resp.Body).Decode(actual)
			require.NoError(t, err)

			require.Equal(t, test.result.Code, actual.Code)
			require.Equal(t, test.result.Message, actual.Message)

			require.Equal(t, test.result.FieldErrors, actual.FieldErrors)
		})
	}
}
