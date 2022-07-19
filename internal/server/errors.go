package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/internal/validate"
)

// sendAPIError translates err into the appropriate HTTP status code, builds a
// response body using api.Error, then sends both as a response to the active
// request.
func sendAPIError(c *gin.Context, err error) {
	resp := &api.Error{
		Code:    http.StatusInternalServerError,
		Message: "internal server error", // don't leak any info by default
	}

	var validationError validate.Error
	var uniqueConstraintError data.UniqueConstraintError
	var authzError access.AuthorizationError

	log := logging.L.Debug()

	switch {
	case errors.Is(err, internal.ErrUnauthorized):
		resp.Code = http.StatusUnauthorized
		// hide the error text, it may contain sensitive information
		resp.Message = "unauthorized"
		// log the error at info because it is not in the response
		log = logging.L.Info()

	case errors.Is(err, data.ErrAccessKeyExpired):
		resp.Code = http.StatusUnauthorized
		// this means the key was once valid, so include some extra details
		resp.Message = fmt.Sprintf("%s: %s", internal.ErrUnauthorized, err)

	case errors.As(err, &authzError):
		resp.Code = http.StatusForbidden
		resp.Message = authzError.Error()

	case errors.As(err, &uniqueConstraintError):
		resp.Code = http.StatusConflict
		resp.Message = err.Error()

	case errors.Is(err, internal.ErrNotFound):
		resp.Code = http.StatusNotFound
		resp.Message = err.Error()

	case errors.As(err, &validationError):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()
		for name, problems := range validationError {
			resp.FieldErrors = append(resp.FieldErrors, api.FieldError{
				FieldName: name,
				Errors:    problems,
			})
		}
		sort.Slice(resp.FieldErrors, func(i, j int) bool {
			return resp.FieldErrors[i].FieldName < resp.FieldErrors[j].FieldName
		})

	case errors.Is(err, internal.ErrBadRequest), errors.Is(err, providers.ErrValidation):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()

	case errors.Is(err, internal.ErrNotImplemented):
		resp.Code = http.StatusNotImplemented
		resp.Message = internal.ErrNotImplemented.Error()

	case errors.Is(err, internal.ErrBadGateway):
		resp.Code = http.StatusBadGateway
		resp.Message = err.Error()

	case errors.Is(err, context.DeadlineExceeded):
		resp.Code = http.StatusGatewayTimeout // not ideal, but StatusRequestTimeout isn't intended for this.
		resp.Message = "request timed out"

	default:
		log = logging.L.Error()
	}

	log.CallerSkipFrame(1).
		Err(err).
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Int32("statusCode", resp.Code).
		Str("remoteAddr", c.Request.RemoteAddr).
		Msg("api request error")

	c.JSON(int(resp.Code), resp)
	c.Abort()
}
