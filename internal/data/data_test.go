package data

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
)

func setup(t *testing.T) *gorm.DB {
	driver, err := NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := NewDB(driver)
	require.NoError(t, err)

	logging.L = zaptest.NewLogger(t)
	logging.S = logging.L.Sugar()

	return db
}

func TestID(t *testing.T) {
	id := NewID()

	require.Equal(t, id.Version().String(), "VERSION_1")
	require.Equal(t, id.Variant().String(), "RFC4122")
}
