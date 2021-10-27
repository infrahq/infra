package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConfigDefault(t *testing.T) {
	logger, _ := Initialize("")

	assert.NotNil(t, logger)

	if checked := logger.Check(zap.InfoLevel, "default"); checked == nil {
		assert.Fail(t, "could not log info level messages")
	}

	if checked := logger.Check(zap.DebugLevel, "not default"); checked != nil {
		assert.Fail(t, "should not log debug level messages")
	}
}

func TestConfigValidLevel(t *testing.T) {
	logger, _ := Initialize("debug")

	assert.NotNil(t, logger)

	if checked := logger.Check(zap.DebugLevel, "not default"); checked == nil {
		assert.Fail(t, "could not log debug level messages")
	}
}

func TestConfigInvalidLevel(t *testing.T) {
	logger, err := Initialize("invalid")

	assert.Nil(t, logger)
	assert.NotNil(t, err)
}
