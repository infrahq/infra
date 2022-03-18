package server

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAPIGen(t *testing.T) {
	// must run from infra root dir
	wd, err := os.Getwd()
	require.NoError(t, err)

	parts := strings.Split(wd, string(os.PathSeparator))

	if parts[len(parts)-1] != "infra" {
		err := os.Chdir("../..")
		require.NoError(t, err)
	}

	s := &Server{}
	s.GenerateRoutes()
}
