// Package logging provides a shared logger and log utilities to be used in all internal packages.
package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

var (
	L, _                              = Initialize(int(zap.InfoLevel))
	S             *zap.SugaredLogger  = L.Sugar()
	defaultWriter zapcore.WriteSyncer = os.Stderr
)

func Initialize(v int) (*zap.Logger, error) {
	atom := zap.NewAtomicLevelAt(zapcore.Level(-v))
	writer := zapcore.Lock(filtered(defaultWriter))

	var encoder zapcore.Encoder

	if term.IsTerminal(int(os.Stdin.Fd())) {
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

// TODO: Remove the filtered writer after Go stops writing header
// values to errors, as it's cpu expensive to search every log line.
// https://groups.google.com/g/golang-codereviews/c/BOSa6DE8tnI
func filtered(logger zapcore.WriteSyncer) zapcore.WriteSyncer {
	return &filteredWriterSyncer{
		dest: logger,
	}
}

type filteredWriterSyncer struct {
	dest zapcore.WriteSyncer
}

func (w *filteredWriterSyncer) Write(b []byte) (int, error) {
	if idx := bytes.Index(b, []byte("invalid header field value")); idx >= 0 {
		idx += 26 // len("invalid header field value")
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
			// on error write nothing, just to be safe.
			return 0, nil
		}
		if msg, ok := m["msg"]; ok {
			if smsg, ok := msg.(string); ok {
				if idx := strings.Index(smsg, "invalid header field value"); idx >= 0 {
					m["msg"] = smsg[:idx+26] // len("invalid header field value")
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
