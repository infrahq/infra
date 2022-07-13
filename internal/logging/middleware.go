package logging

import (
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
		log := L.With().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("host", c.Request.Host).
			Str("remoteAddr", c.Request.RemoteAddr).
			Str("userAgent", c.Request.UserAgent()).
			Int64("contentLength", c.Request.ContentLength).
			Logger()

		begin := time.Now()

		c.Next()

		level := zerolog.InfoLevel
		if len(c.Errors) > 0 {
			level = zerolog.ErrorLevel
		}

		errs := make([]error, 0, len(c.Errors))
		for _, err := range c.Errors {
			errs = append(errs, err.Err)
		}

		// attach log sampler. should not sample logs if level >= Warn
		if level <= zerolog.InfoLevel {
			log = log.Sample(sampler.Get(c.Request.Method, c.FullPath()))
		}

		log.WithLevel(level).
			Errs("errors", errs).
			Dur("elapsed", time.Since(begin)).
			Int("statusCode", c.Writer.Status()).
			Int("size", c.Writer.Size()).
			Msg("")
	}
}
