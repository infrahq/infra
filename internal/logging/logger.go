// Package logging provides a shared logger and log utilities to be used in all internal packages.
package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"

	"github.com/infrahq/infra/internal/server/models"
)

var (
	level                    = zap.NewAtomicLevel()
	L                        = newLogger(level, os.Stderr)
	S     *zap.SugaredLogger = L.Sugar()
)

// SetLevel of the global loggers L and S.
func SetLevel(l string) error {
	return level.UnmarshalText([]byte(l))
}

func newLogger(level zapcore.LevelEnabler, stderr zapcore.WriteSyncer) *zap.Logger {
	writer := zapcore.Lock(filtered(stderr))
	levelEncoder := zapcore.CapitalLevelEncoder
	if term.IsTerminal(int(os.Stdin.Fd())) {
		levelEncoder = zapcore.CapitalColorLevelEncoder
	}
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		LevelKey:         "level",
		EncodeLevel:      levelEncoder,
		MessageKey:       "message",
		ConsoleSeparator: "  ",
	})
	return zap.New(zapcore.NewCore(encoder, writer, level), zap.AddCaller())
}

// SetServerLogger changes L and S to a logger that is appropriate for long
// running processes like the api server and connectors. The logger uses
// json format and includes the function name and line number in the log message.
//
// SetServerLogger should not be called concurrently. It should be called
// before any goroutines that may use the logger are started.
func SetServerLogger() {
	L = newServerLogger(level, os.Stdout, os.Stderr)
	S = L.Sugar()
}

func newServerLogger(level zapcore.LevelEnabler, stdout, stderr zapcore.WriteSyncer) *zap.Logger {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		writer := zapcore.Lock(filtered(stderr))
		encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalColorLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		})

		return zap.New(zapcore.NewCore(encoder, writer, level), zap.AddCaller())
	}

	return zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.Lock(filtered(stdout)),
			level,
		),
		zap.AddCaller(),
	)
}

func StandardErrorLog() *log.Logger {
	errorLog, err := zap.NewStdLogAt(L, zapcore.ErrorLevel)
	if err != nil {
		return nil
	}

	return errorLog
}

// TODO: Remove the filtered writer after Go stops writing header
// values to errors, as it's cpu expensive to search every log line.
// https://github.com/golang/go/pull/48979
func filtered(logger zapcore.WriteSyncer) zapcore.WriteSyncer {
	return &filteredWriterSyncer{
		dest: logger,
	}
}

type filteredWriterSyncer struct {
	dest zapcore.WriteSyncer
}

var strInvalidHeaderFieldValue = []byte("invalid header field value")

func (w *filteredWriterSyncer) Write(b []byte) (int, error) {
	if idx := bytes.Index(b, strInvalidHeaderFieldValue); idx >= 0 {
		idx += len(strInvalidHeaderFieldValue)

		forKeyIdx := bytes.Index(b, []byte("for key"))
		if forKeyIdx > idx {
			return w.dest.Write(append(b[:idx+1], b[forKeyIdx:]...))
		}

		if b[0] != '{' {
			// not json; free to truncate.
			return w.dest.Write(b[:idx])
		}

		// we can't see where the end is. parse the message so you can truncate it. :/
		m := map[string]interface{}{}
		if err := json.Unmarshal(b, &m); err != nil {
			S.Error("Had some trouble parsing log line that needs to be filtered. Omitting log entry")
			// on error write nothing, just to be safe.
			return 0, nil // nolint
		}

		if msg, ok := m["msg"]; ok {
			if smsg, ok := msg.(string); ok {
				if idx := strings.Index(smsg, string(strInvalidHeaderFieldValue)); idx >= 0 {
					m["msg"] = smsg[:idx+len(strInvalidHeaderFieldValue)]

					newBytes, err := json.Marshal(m)
					if err == nil {
						return w.dest.Write(newBytes)
					}
				}
			}
		}
		// write nothing, just to be safe.
		return 0, nil
	}

	return w.dest.Write(b)
}

func (w *filteredWriterSyncer) Sync() error {
	return w.dest.Sync()
}

// Middleware logs incoming requests using configured logger.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		msg := fmt.Sprintf(
			"\"%s %s\" %d %d \"%s\" %s %s %d",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			c.Writer.Size(),
			c.Request.Host,
			c.Request.UserAgent(),
			c.Request.RemoteAddr,
			c.Request.ContentLength,
		)

		logger := L
		// TODO: use access.GetAuthenticatedIdentity, requires refactor
		if raw, ok := c.Get("identity"); ok {
			if identity, ok := raw.(*models.Identity); ok {
				logger = logger.With(zap.Stringer("identity", identity.ID))
			}
		}

		logger.Info(
			msg,
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("statusCode", c.Writer.Status()),
			zap.String("remoteAddr", c.Request.RemoteAddr),
		)
	}
}
