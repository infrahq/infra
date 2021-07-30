package registry

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

func ZapLoggerHttpMiddleware(logger *zap.Logger, next http.Handler) http.HandlerFunc {
	if logger == nil {
		return next.ServeHTTP
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		t1 := time.Now()
		next.ServeHTTP(ww, r)
		logger.Info("finished http method call",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.Status()),
			zap.String("proto", r.Proto),
			zap.Duration("time_ms", time.Since(t1)),
		)
	}
}
