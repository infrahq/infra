package logging

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := L.Sample(zerolog.Often).
			With().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("host", c.Request.Host).
			Str("remoteAddr", c.Request.RemoteAddr).
			Str("userAgent", c.Request.UserAgent()).
			Int64("contentLength", c.Request.ContentLength).
			Logger()

		begin := time.Now()

		c.Next()

		levelName := zerolog.InfoLevel
		if len(c.Errors) > 0 {
			levelName = zerolog.ErrorLevel
		}

		errs := make([]error, 0, len(c.Errors))
		for _, err := range c.Errors {
			errs = append(errs, err.Err)
		}

		log.WithLevel(levelName).
			Errs("errors", errs).
			Dur("elapsed", time.Since(begin)).
			Int("statusCode", c.Writer.Status()).
			Int("size", c.Writer.Size()).
			Msg("")
	}
}
