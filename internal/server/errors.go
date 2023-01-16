package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/redis"
	"github.com/infrahq/infra/internal/validate"
)

// sendAPIError translates err into the appropriate HTTP status code, builds a
// response body using api.Error, then sends both as a response to the active
// request.
func sendAPIError(writer http.ResponseWriter, req *http.Request, err error) {
	resp := &api.Error{
		Code:    http.StatusInternalServerError,
		Message: "internal server error", // don't leak any info by default
	}

	var validationError validate.Error
	var uniqueConstraintError data.UniqueConstraintError
	var overLimitError redis.OverLimitError
	var authnError AuthenticationError
	var apiError api.Error

	log := logging.L.Debug()

	switch {
	case errors.As(err, &apiError):
		// the handler has already created an appropriate error to return
		resp = &apiError

	case errors.Is(err, internal.ErrUnauthorized):
		resp.Code = http.StatusUnauthorized
		// hide the error text, it may contain sensitive information
		resp.Message = "unauthorized"
		// log the error at info because it is not in the response
		log = logging.L.Info()

	case errors.As(err, &authnError):
		resp.Code = http.StatusUnauthorized
		resp.Message = authnError.Message

	case errors.Is(err, data.ErrAccessKeyExpired):
		resp.Code = http.StatusUnauthorized
		// this means the key was once valid, so include some extra details
		resp.Message = fmt.Sprintf("%s: %s", internal.ErrUnauthorized, err)

	case errors.Is(err, access.ErrNotAuthorized):
		resp.Code = http.StatusForbidden
		resp.Message = err.Error()

	case errors.As(err, &uniqueConstraintError):
		*resp = newAPIErrorForUniqueConstraintError(uniqueConstraintError, err.Error())

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

	case errors.Is(err, internal.ErrExpired):
		resp.Code = http.StatusGone
		resp.Message = "requested resource has expired"

	case errors.Is(err, internal.ErrBadRequest):
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()

	case errors.Is(err, internal.ErrNotModified):
		resp.Code = http.StatusNotModified
		resp.Message = err.Error()

	case errors.Is(err, internal.ErrBadGateway):
		resp.Code = http.StatusBadGateway
		resp.Message = err.Error()

	case errors.As(err, &overLimitError):
		writer.Header().Set("Retry-After", strconv.Itoa(int(overLimitError.RetryAfter.Seconds())))
		resp.Code = http.StatusTooManyRequests
		resp.Message = err.Error()

	case errors.Is(err, context.DeadlineExceeded):
		resp.Code = http.StatusGatewayTimeout // not ideal, but StatusRequestTimeout isn't intended for this.
		resp.Message = "request timed out"

	case errors.Is(err, context.Canceled):
		// Nginx uses this non-standard error code for the same purpose
		resp.Code = 499
		resp.Message = fmt.Sprintf("client closed the request: %v", err)

	default:
		log = logging.L.Error()
	}

	log.CallerSkipFrame(1).
		Err(err).
		Str("method", req.Method).
		Str("path", req.URL.Path).
		Int32("statusCode", resp.Code).
		Str("remoteAddr", req.RemoteAddr).
		Msg("api request error")

	if resp.Code == http.StatusNotModified {
		writer.WriteHeader(int(resp.Code))
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(int(resp.Code))
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		logging.L.Error().Err(err).Msg("failed to write error response")
	}
}

func newAPIErrorForUniqueConstraintError(ucErr data.UniqueConstraintError, msg string) api.Error {
	apiError := api.Error{
		Code:    http.StatusConflict,
		Message: msg,
	}
	apiError.FieldErrors = []api.FieldError{{
		FieldName: ucErr.Column,
		Errors:    []string{ucErr.Error()},
	}}
	return apiError
}

// AuthenticationError is used to respond with a 401 Unauthorized response code.
// Unlike internal.ErrUnauthorized, AuthenticationError includes an error message
// in the response.
type AuthenticationError struct {
	// Message is sent as the api.Error.Message in the response. Message should
	// always be a hard coded string to ensure that it does not include any
	// sensitive data.
	Message string
}

func (a AuthenticationError) Error() string {
	return a.Message
}
