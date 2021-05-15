package data

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewData(t *testing.T) {
	td, err := ioutil.TempDir("", "db")
	require.NoError(t, err)

	data, err := NewData(td)
	assert.NoError(t, err)
	assert.IsType(t, &Data{}, data)
}
