package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
)

func (a *API) sendAPIError(c *gin.Context, err error) {
	resp := &api.Error{
		Code:    http.StatusInternalServerError,
		Message: "internal server error", // don't leak any info by default
	}

	validationErrors := &validator.ValidationErrors{}

	switch {
	case errors.Is(err, internal.ErrUnauthorized):
		resp.Code = http.StatusUnauthorized
		resp.Message = "unauthorized"
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.Is(err, internal.ErrForbidden):
		resp.Code = http.StatusForbidden
		resp.Message = "forbidden"
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.Is(err, internal.ErrDuplicate):
		resp.Code = http.StatusConflict
		resp.Message = err.Error()
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.Is(err, internal.ErrNotFound):
		resp.Code = http.StatusNotFound
		resp.Message = err.Error()
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.As(err, validationErrors):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()

		parseFieldErrors(resp, validationErrors)

		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.Is(err, internal.ErrBadRequest):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.Is(err, internal.ErrNotImplemented):
		resp.Code = http.StatusNotImplemented
		resp.Message = internal.ErrNotImplemented.Error()
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.Is(err, internal.ErrBadGateway):
		resp.Code = http.StatusBadGateway
		resp.Message = err.Error()
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	case errors.Is(err, (*validator.InvalidValidationError)(nil)):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()
		logging.WrappedSugarLogger(c).Debugw(err.Error(), "statusCode", resp.Code)
	default:
		logging.WrappedSugarLogger(c).Errorw(err.Error(), "statusCode", resp.Code)
	}

	a.t.Event(c, "error", Properties{"code": resp.Code})

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
