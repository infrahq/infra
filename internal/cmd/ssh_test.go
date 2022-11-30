package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server"
)

func TestSSHHostsCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("INFRA_LOG_LEVEL", "debug")

	srvDir := t.TempDir()
	opts := defaultServerOptions(srvDir)
	opts.Config = server.Config{
		Users: []server.User{
			{Name: "admin@example.com", AccessKey: "0000000001.adminadminadminadmin1234"},
			{Name: "anyuser@example.com", AccessKey: "0000000002.notadminsecretnotadmin02"},
		},
		Grants: []server.Grant{
			{User: "admin@example.com", Resource: "infra", Role: "admin"},
			{User: "anyuser@example.com", Resource: "prodhost", Role: "connect"},
		},
	}
	setupServerOptions(t, &opts)
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx := context.Background()
	runAndWait(ctx, t, srv.Run)

	client, err := NewAPIClient(&APIClientOpts{
		AccessKey: "0000000001.adminadminadminadmin1234",
		Host:      srv.Addrs.HTTPS.String(),
		Transport: httpTransportForHostConfig(&ClientHostConfig{TrustedCertificate: string(opts.TLS.Certificate)}),
	})
	assert.NilError(t, err)

	hostKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDbFeHekEHKkH8R9UgQt586OjjYRAC2FnxxWA8+T68wm4XB5Rcvrth2hAZhN12NKOaR57MuscFkmn6fUc2Z+hjj8saX14/zyuJLqz2Svk9p2tFkVM+A1B1ZuluZLyGe86i5anb+H19L2MkcojMCxSgScwgRgRmcXpTEQAIILQL5HCfxtMpFmmL03Vl1sPWh9p7G8FPuqMTElSITumMokrMFeDEP8H06LhjM4jBgGxZds7FVFqKrQgE73GdGKl936HgY9JE8RLSOyJ2GSVcKUZKYirYCa6LHAO37NSA0ulZrlx0nl6Yt7nMIAhDDRY/UzeJ6wW1+UAzmqA1mbY16+AHY8ItIQCkfxXaoS59Z3k7qczfYTpZpNYeNo/7Xypt1yz0NRdKuDzGLhzYIg2toQ3jSXWWgUlJ8IGnrvbStvt+tnnbObpgkVBRHZCIUUXB/ZC5zDXvGy+5BTkTPSSyXes39ZwRgAeI0OHO1Fr8gvxscWWB2ygfPRmpLo5JWOLABxi8= root@006100a32148\n"
	_, err = client.CreateDestination(ctx, &api.CreateDestinationRequest{
		Name: "prodhost",
		Kind: "ssh",
		Connection: api.DestinationConnection{
			URL: "127.12.12.1",
			CA:  api.PEM(hostKey),
		},
	})
	assert.NilError(t, err)

	users, err := client.ListUsers(ctx, api.ListUsersRequest{Name: "anyuser@example.com"})
	assert.NilError(t, err)
	assert.Equal(t, len(users.Items), 1)
	user := users.Items[0]
	assert.Equal(t, user.SSHLoginName, "anyuser")

	cfg := newTestClientConfigForServer(srv, user, "0000000002.notadminsecretnotadmin02")
	assert.NilError(t, writeConfig(&cfg))

	err = Run(ctx, "ssh", "hosts", "127.12.12.1", "22")
	assert.NilError(t, err)

	expected := fs.Expected(t,
		// the mode of the temp dir is not relevant to this test
		fs.MatchAnyFileMode,
		// the infra dir is not relevant to this test
		fs.WithDir(".infra", fs.MatchExtraFiles),
		fs.WithDir(".ssh",
			fs.WithMode(0o700),
			fs.WithDir("infra",
				fs.WithMode(0o700),
				fs.WithFile("config", `

# This file is managed by Infra. Do not edit!

Match 127.12.12.1
    IdentityFile ~/.ssh/infra/key
    IdentitiesOnly yes
    UserKnownHostsFile ~/.ssh/infra/known_hosts
    User anyuser
    Port 22

`,
					fs.WithMode(0o600)),
				fs.WithFile("key", "",
					fs.WithMode(0o600),
					fs.MatchAnyFileContent),
				fs.WithFile("key.pub", "",
					fs.WithMode(0o600),
					fs.MatchAnyFileContent),
				fs.WithFile("known_hosts",
					"127.12.12.1 "+hostKey,
					fs.WithMode(0o600)),
			),
		),
	)
	assert.Assert(t, fs.Equal(home, expected))

	raw, err := os.ReadFile(filepath.Join(home, ".ssh/infra/key"))
	assert.NilError(t, err)

	_, err = ssh.ParseRawPrivateKey(raw)
	assert.NilError(t, err)

	raw, err = os.ReadFile(filepath.Join(home, ".ssh/infra/key.pub"))
	assert.NilError(t, err)

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	assert.NilError(t, err)
	assert.Equal(t, pubKey.Type(), "ssh-rsa")

	updated, err := client.GetUser(ctx, user.ID)
	assert.NilError(t, err)

	assert.Equal(t, len(updated.PublicKeys), 1)
	assert.Equal(t, updated.PublicKeys[0].KeyType, "ssh-rsa")
	parts := strings.Fields(string(raw))
	assert.Equal(t, updated.PublicKeys[0].PublicKey, parts[1])
}

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
		err := updateUserSSHConfig(newCLI(ctx))
		assert.NilError(t, err)

		expectedOutput := "has been updated to connect to Infra SSH destinations"
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
		assert.Equal(t, fi.Mode(), os.FileMode(0o600))

		tc.expected(t, fh)
	}

	contentWithMatchLine := `

Host somethingelse

Match something


Match exec "infra ssh hosts %h"
    Include somethingelse


Host more below
`

	contentNoMatchLine := `
Host bastion
	Username shared
`

	expectedInfraSSHConfig := `

Match exec "infra ssh hosts %h %p"
    Include ~/.ssh/infra/config

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
				assert.NilError(t, os.MkdirAll(filepath.Dir(filename), 0o700))
				err := os.WriteFile(filename, []byte(contentWithMatchLine), 0o600)
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
				assert.NilError(t, os.MkdirAll(filepath.Dir(filename), 0o700))
				err := os.WriteFile(filename, []byte(contentNoMatchLine), 0o600)
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
