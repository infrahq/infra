// Package logging provides a shared logger and log utilities to be used in all internal packages.
package logging

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

var L *zap.Logger // L is the default zap logger initialized at start-up

func init() {
	var err error

	L, err = Build()
	if err != nil {
		panic(err)
	}
}

// Build makes a new production Zap logger with the configured log level.
func Build() (*zap.Logger, error) {
	return config(os.Getenv("INFRA_LOG_LEVEL")).Build()
}

func config(lvl string) zap.Config {
	conf := zap.NewProductionConfig()
	atomicLvl := zap.NewAtomicLevel()

	err := atomicLvl.UnmarshalText([]byte(lvl))
	if err != nil {
		fmt.Printf("Using default log level. %v\n", err)
	} else {
		conf.Level = atomicLvl
	}

	return conf
}
