// Package logging provides a shared logger and log utilities to be used in all internal packages.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/term"
)

var (
	L = newLogger(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
)

type logger struct {
	zerolog.Logger
}

func init() {
	zerolog.DisableSampling(true)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		short := filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file))
		return fmt.Sprintf("%s:%d", short, line)
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func newLogger(writer io.Writer) *logger {
	return &logger{
		Logger: zerolog.New(writer).With().Timestamp().Caller().Logger(),
	}
}

func UseServerLogger() {
	// If the server is run from an interactive terminal, use the default ConsoleWriter
	if os.Stdin != nil && term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}
	L = newLogger(os.Stderr)
}

func Tracef(format string, v ...interface{}) {
	L.Trace().Msgf(format, v...)
}

func Debugf(format string, v ...interface{}) {
	L.Debug().Msgf(format, v...)
}

func Infof(format string, v ...interface{}) {
	L.Info().Msgf(format, v...)
}

func Warnf(format string, v ...interface{}) {
	L.Warn().Msgf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	L.Error().Msgf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	L.Fatal().Msgf(format, v...)
}

func Panicf(format string, v ...interface{}) {
	L.Panic().Msgf(format, v...)
}

func SetLevel(levelName string) error {
	level, err := zerolog.ParseLevel(levelName)
	if err != nil {
		return err
	}

	zerolog.SetGlobalLevel(level)
	return nil
}

type TestingT interface {
	zerolog.TestingLog
	Cleanup(func())
}

// PatchLogger sets the global L logger to write logs to t. When the test ends
// the global L logger is reset to the previous value.
// PatchLogger changes a static variable, so tests that use PatchLogger can not
// use t.Parallel.
func PatchLogger(t TestingT) {
	origL := L
	L = newLogger(zerolog.NewTestWriter(t))
	t.Cleanup(func() {
		L = origL
	})
}
