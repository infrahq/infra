package logging

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Sampler struct {
	fn       func() zerolog.Sampler
	samplers sync.Map
}

func NewSampler(fn func() zerolog.Sampler) *Sampler {
	return &Sampler{fn: fn}
}

func (c *Sampler) Get(fields ...string) zerolog.Sampler {
	key := strings.Join(fields, "-")
	raw, ok := c.samplers.Load(key)
	if !ok {
		// Only use LoadOrStore on a failed load, to avoid creating unnecessary samplers
		raw, _ = c.samplers.LoadOrStore(key, c.fn())
	}

	return raw.(zerolog.Sampler) // nolint:forcetypeassert
}

func Middleware() gin.HandlerFunc {
	sampler := NewSampler(func() zerolog.Sampler {
		return &zerolog.BurstSampler{
			Burst:  1,
			Period: 7 * time.Second,
		}
	})

	return func(c *gin.Context) {
		method := c.Request.Method
		log := L.With().
			Str("method", method).
			Str("path", c.Request.URL.Path).
			Str("localAddr", c.Request.Host).
			Str("remoteAddr", c.Request.RemoteAddr).
			Str("userAgent", c.Request.UserAgent()).
			Int64("contentLength", c.Request.ContentLength).
			Logger()

		begin := time.Now()

		c.Next()

		status := c.Writer.Status()

		// sample logs for successful GET request if the log level is INFO or above
		if status < 400 && method == http.MethodGet && zerolog.GlobalLevel() >= zerolog.InfoLevel {
			log = log.Sample(sampler.Get(c.Request.Method, c.FullPath()))
		}

		log.Info().
			Dur("elapsed", time.Since(begin)).
			Int("statusCode", status).
			Int("size", c.Writer.Size()).
			Msg("")
	}
}
