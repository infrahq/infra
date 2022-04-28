package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
)

// sendAPIError translates err into the appropriate HTTP status code, builds a
// response body using api.Error, then sends both as a response to the active
// request.
func (a *API) sendAPIError(c *gin.Context, err error) {
	resp := &api.Error{
		Code:    http.StatusInternalServerError,
		Message: "internal server error", // don't leak any info by default
	}

	validationErrors := &validator.ValidationErrors{}

	log := logging.L.WithOptions(zap.AddCallerSkip(1)).Debug

	switch {
	case errors.Is(err, internal.ErrUnauthorized):
		resp.Code = http.StatusUnauthorized
		// hide the error text, it may contain sensitive information
		resp.Message = "unauthorized"
	case errors.Is(err, internal.ErrForbidden):
		resp.Code = http.StatusForbidden
		// hide the error text, it may contain sensitive information
		resp.Message = "forbidden"
	case errors.Is(err, internal.ErrDuplicate):
		resp.Code = http.StatusConflict
		resp.Message = err.Error()
	case errors.Is(err, internal.ErrNotFound):
		resp.Code = http.StatusNotFound
		resp.Message = err.Error()
	case errors.As(err, validationErrors):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()

		parseFieldErrors(resp, validationErrors)
	case errors.Is(err, internal.ErrBadRequest), errors.Is(err, authn.ErrValidation):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()
	case errors.Is(err, internal.ErrNotImplemented):
		resp.Code = http.StatusNotImplemented
		resp.Message = internal.ErrNotImplemented.Error()
	case errors.Is(err, internal.ErrBadGateway):
		resp.Code = http.StatusBadGateway
		resp.Message = err.Error()
	case errors.Is(err, (*validator.InvalidValidationError)(nil)):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()
	case errors.Is(err, context.DeadlineExceeded):
		resp.Code = http.StatusGatewayTimeout // not ideal, but StatusRequestTimeout isn't intended for this.
		resp.Message = "request timed out"
	default:
		log = logging.L.WithOptions(zap.AddCallerSkip(1)).Error
	}

	log("api request error", zap.Error(err), zap.Int32("statusCode", resp.Code))

	if resp.Code >= 500 {
		a.t.Event(c, "error", Properties{
			"code": resp.Code,
			"path": c.FullPath(),
		})
	}

	c.JSON(int(resp.Code), resp)
	c.Abort()
}

func parseFieldErrors(resp *api.Error, validationErrors *validator.ValidationErrors) {
	errs := map[string][]string{}

	for _, field := range *validationErrors {
		msg := ""
		if field.Tag() == "required" {
			msg = "is required"
		} else {
			msg = fmt.Sprintf("failed the %q check", field.Tag())
		}

		errs[field.Field()] = append(errs[field.Field()], msg)
	}

	for f, vals := range errs {
		resp.FieldErrors = append(resp.FieldErrors, api.FieldError{FieldName: f, Errors: vals})
	}

	// rebuild the error message, because the default is just bad.
	if len(resp.FieldErrors) > 0 {
		errs := []string{}
		for _, fe := range resp.FieldErrors {
			errs = append(errs, fe.FieldName+": "+strings.Join(fe.Errors, ", "))
		}

		resp.Message = strings.Join(errs, ". ")
	}
}

func removed(version string) func(c *gin.Context) {
	return func(c *gin.Context) {
		msg := fmt.Sprintf("This API endpoint was removed in version %v. Please upgrade your client.", version)
		resp := &api.Error{Code: http.StatusGone, Message: msg}
		c.JSON(int(resp.Code), resp)
		c.Abort()
	}
}
