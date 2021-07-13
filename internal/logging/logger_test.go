package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConfigDefault(t *testing.T) {
	got := config("") // the empty string here models what Build() would pass into this function when there is no log level env var
	assert.Equal(t, zap.NewProductionConfig().Level.String(), got.Level.String())
}

func TestConfigValidLevel(t *testing.T) {
	got := config("debug")
	assert.Equal(t, "debug", got.Level.String())
}

func TestConfigInvalidLevel(t *testing.T) {
	got := config("invalid") // invalid is not a level declared in zap levels
	assert.Equal(t, zap.NewProductionConfig().Level.String(), got.Level.String())
}
