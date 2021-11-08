// Package logging provides a shared logger and log utilities to be used in all internal packages.
package logging

import (
	"io"
	"log"
	"os"

	"github.com/gorilla/handlers"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

var (
	L *zap.Logger        = zap.L()
	S *zap.SugaredLogger = zap.S()
)

func Initialize(v int) (*zap.Logger, error) {
	atom := zap.NewAtomicLevelAt(zapcore.Level(-v))

	var (
		encoder zapcore.Encoder
		writer  zapcore.WriteSyncer
	)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		writer = zapcore.Lock(os.Stderr)
		encoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalColorLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		})
	} else {
		writer = zapcore.Lock(os.Stdout)
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}

	core := zapcore.NewCore(encoder, writer, atom)

	return zap.New(core, zap.AddCaller()), nil
}

func ZapLogFormatter(_ io.Writer, params handlers.LogFormatterParams) {
	L.Debug("handled request",
		zap.String("method", params.Request.Method),
		zap.String("path", params.URL.Path),
		zap.Int("status", params.StatusCode),
		zap.Int("size", params.Size))
}

func StandardErrorLog() *log.Logger {
	errorLog, err := zap.NewStdLogAt(L, zapcore.ErrorLevel)
	if err != nil {
		return nil
	}

	return errorLog
}
