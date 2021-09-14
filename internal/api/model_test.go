package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanupDomain(t *testing.T) {
	actual := cleanupDomain("dev123123-admin.okta.com ")
	expected := "dev123123.okta.com"
	require.Equal(t, expected, actual)
}
