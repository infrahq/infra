package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	bolt "github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func NewTestDB(t *testing.T) *bolt.DB {
	t.Helper()

	td, err := ioutil.TempDir("", "db")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(td) })

	db, err := bolt.Open(filepath.Join(td, "test.db"), 0600, nil)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	return db
}

func TestRandStringNegativeLen(t *testing.T) {
	assert.Equal(t, randString(-1), "")
}

func TestRandStringLen(t *testing.T) {
	assert.Equal(t, len(randString(20)), 20)
}

func TestNewData(t *testing.T) {
	td, err := ioutil.TempDir("", "db")
	require.NoError(t, err)

	data, err := NewData(td)
	assert.NoError(t, err)
	assert.IsType(t, &Data{}, data)
}
