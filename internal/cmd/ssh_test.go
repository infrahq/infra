package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestUpdateUserSSHConfig(t *testing.T) {
	type testCase struct {
		name            string
		setup           func(t *testing.T, filename string)
		expected        func(t *testing.T, fh *os.File)
		expectCLIOutput bool
	}

	run := func(t *testing.T, tc testCase) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)

		sshConfigFilename := filepath.Join(home, ".ssh/config")

		if tc.setup != nil {
			tc.setup(t, sshConfigFilename)
		}

		ctx := context.Background()
		ctx, bufs := PatchCLI(ctx)
		err := updateUserSSHConfig(newCLI(ctx), "theusername")
		assert.NilError(t, err)

		expectedOutput := "has been created or updated to use 'infra ssh hosts'"
		if tc.expectCLIOutput {
			assert.Assert(t, cmp.Contains(bufs.Stdout.String(), expectedOutput))
		} else {
			assert.Equal(t, bufs.Stdout.String(), "")
		}

		fh, err := os.Open(sshConfigFilename)
		assert.NilError(t, err)
		defer fh.Close()

		fi, err := fh.Stat()
		assert.NilError(t, err)
		assert.Equal(t, fi.Mode(), os.FileMode(0600))

		tc.expected(t, fh)
	}

	var contentWithMatchLine = `

Host somethingelse

Match something


Match exec "infra ssh hosts %h"
    ProxyCommand "infra ssh connect %h"


Host more below
`

	var contentNoMatchLine = `
Host bastion
	Username shared
`

	var expectedInfraSSHConfig = `

Match exec "infra ssh hosts %h"
    IdentityFile ~/.ssh/infra/key
    IdentitiesOnly yes
    User theusername
    UserKnownHostsFile ~/.ssh/infra/known_hosts

`

	testCases := []testCase{
		{
			name:            "file does not exist",
			expectCLIOutput: true,
			expected: func(t *testing.T, fh *os.File) {
				content, err := io.ReadAll(fh)
				assert.NilError(t, err)
				assert.Equal(t, string(content), expectedInfraSSHConfig)
			},
		},
		{
			name: "file exists with match line",
			setup: func(t *testing.T, filename string) {
				assert.NilError(t, os.MkdirAll(filepath.Dir(filename), 0700))
				err := os.WriteFile(filename, []byte(contentWithMatchLine), 0600)
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, fh *os.File) {
				content, err := io.ReadAll(fh)
				assert.NilError(t, err)
				assert.Equal(t, string(content), contentWithMatchLine)
			},
		},
		{
			name: "file exists with no match line",
			setup: func(t *testing.T, filename string) {
				assert.NilError(t, os.MkdirAll(filepath.Dir(filename), 0700))
				err := os.WriteFile(filename, []byte(contentNoMatchLine), 0600)
				assert.NilError(t, err)
			},
			expectCLIOutput: true,
			expected: func(t *testing.T, fh *os.File) {
				content, err := io.ReadAll(fh)
				assert.NilError(t, err)
				expected := contentNoMatchLine + expectedInfraSSHConfig
				assert.Equal(t, string(content), expected)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestHasInfraMatchLine(t *testing.T) {
	type testCase struct {
		name     string
		input    io.Reader
		expected bool
	}

	run := func(t *testing.T, tc testCase) {
		actual := hasInfraMatchLine(tc.input)
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:     "nil",
			expected: false,
		},
		{
			name:     "empty file",
			input:    strings.NewReader(""),
			expected: false,
		},
		{
			name: "only comments",
			input: strings.NewReader(`

# Match exec "infra ssh hosts %h"
# Other comments

`),
		},
		{
			name: "different match lines",
			input: strings.NewReader(`
Match "infra ssh hosts"

Match hostname.example.com

`),
		},
		{
			name: "Match line found",
			input: strings.NewReader(`

Match other lines before

# Some comment
MATCH exec "infra ssh hosts %h"
	Anything here

Match other lines after

`),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
