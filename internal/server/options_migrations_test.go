package server

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestApplyVersion0ConfigToLatest(t *testing.T) {
	content := `
                    identities:
                      - name: walter`

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("cfg.yaml", content))

	options := &Options{}
	err := ApplyOptions(options, dir.Join("cfg.yaml"), &pflag.FlagSet{})
	assert.NilError(t, err)

	expected := &Options{
		Version: 0.3,
	}
	expected.Users = []User{
		{
			Name: "walter",
		},
	}

	assert.DeepEqual(t, options, expected)
}

func TestApplyVersion02ConfigToLatest(t *testing.T) {
	content := `
                    sessionExtensionDeadline: 12h0m0s`

	dir := fs.NewDir(t, t.Name(),
		fs.WithFile("cfg.yaml", content))

	options := &Options{}
	err := ApplyOptions(options, dir.Join("cfg.yaml"), &pflag.FlagSet{})
	assert.NilError(t, err)

	expected := &Options{
		Version: 0.3,
	}
	expected.SessionInactivityTimeout = 12 * time.Hour

	assert.DeepEqual(t, options, expected)
}
