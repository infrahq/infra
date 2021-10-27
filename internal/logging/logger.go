// Package logging provides a shared logger and log utilities to be used in all internal packages.
package logging

import (
	"go.uber.org/zap"
)

var (
	L *zap.Logger        = zap.L()
	S *zap.SugaredLogger = zap.S()
)

func Initialize(level string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	atom := zap.NewAtomicLevel()

	err := atom.UnmarshalText([]byte(level))
	if err != nil {
		return nil, err
	}

	config.Level = atom

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
