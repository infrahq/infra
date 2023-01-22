package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/infrahq/infra/internal/logging"
)

type logSampler struct {
	fn       func() zerolog.Sampler
	samplers sync.Map
}

func newLogSampler(fn func() zerolog.Sampler) *logSampler {
	return &logSampler{fn: fn}
}

func (c *logSampler) Get(fields ...string) zerolog.Sampler {
	key := strings.Join(fields, "-")
	raw, ok := c.samplers.Load(key)
	if !ok {
		// Only use LoadOrStore on a failed load, to avoid creating unnecessary samplers
		raw, _ = c.samplers.LoadOrStore(key, c.fn())
	}

	return raw.(zerolog.Sampler) // nolint:forcetypeassert
}

func loggingMiddleware(enableSampling bool) gin.HandlerFunc {
	sampler := newLogSampler(func() zerolog.Sampler {
		return &zerolog.BurstSampler{
			Burst:  1,
			Period: 7 * time.Second,
		}
	})

	return func(c *gin.Context) {
		begin := time.Now()
		c.Next()

		method := c.Request.Method
		status := c.Writer.Status()
		logger := logging.L.Logger

		// sample logs for successful GET request if the log level is INFO or above
		if enableSampling && status < 400 && method == http.MethodGet && zerolog.GlobalLevel() >= zerolog.InfoLevel {
			logger = logger.Sample(sampler.Get(c.Request.Method, c.FullPath()))
		}

		event := logger.Info().
			Str("method", method).
			Str("path", c.Request.URL.Path).
			Str("localAddr", c.Request.Host).
			Str("remoteAddr", c.ClientIP()).
			Str("userAgent", c.Request.UserAgent())

		if c.Request.ContentLength > 0 {
			event = event.Int64("contentLength", c.Request.ContentLength)
		}

		if user := rCtx.Authenticated.User; user != nil {
			event = event.Str("userID", user.ID.String())
		} else if rCtx.Response != nil && rCtx.Response.LoginUserID != 0 {
			event = event.Str("userID", rCtx.Response.LoginUserID.String())
		}

		if org := rCtx.Authenticated.Organization; org != nil {
			event = event.Str("orgID", org.ID.String())
		} else if rCtx.Response != nil && rCtx.Response.SignupOrgID != 0 {
			event = event.Str("orgID", rCtx.Response.SignupOrgID.String())
		}

		rCtx.Response.ApplyLogFields(event)

		event.Dur("elapsed", time.Since(begin)).
			Int("statusCode", status).
			Int("size", c.Writer.Size()).
			Msg("API request completed")
	}
}
