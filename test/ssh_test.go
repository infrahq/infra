package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/poll"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

var infraServerURL = "https://localhost:4443"
var infraServerCAFile = "../internal/server/testdata/pki/ca.crt"

func TestSSHDestination(t *testing.T) {
	adminClient := api.Client{
		Name:      "testing",
		Version:   "0.0.1",
		URL:       infraServerURL,
		AccessKey: "aaaaaaaaaa.000000000000000000000000",
		HTTP: http.Client{
			Transport: httpTransport(t, infraServerCAFile),
			Timeout:   time.Minute,
		},
	}

	t.Run("ubuntu", func(t *testing.T) {
		testSSHDestination(t, testCase{
			adminClient: adminClient,
			destination: "ubuntu",
			port:        "8220",
		})
	})

	t.Run("debian", func(t *testing.T) {
		testSSHDestination(t, testCase{
			adminClient: adminClient,
			destination: "debian",
			port:        "8221",
		})
	})

	t.Run("redhat", func(t *testing.T) {
		testSSHDestination(t, testCase{
			adminClient: adminClient,
			destination: "redhat",
			port:        "8222",
		})
	})
}

type testCase struct {
	adminClient api.Client
	destination string
	port        string
}

func testSSHDestination(t *testing.T, tc testCase) {
	ctx := context.Background()
	adminClient := tc.adminClient

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		dests, err := adminClient.ListDestinations(ctx,
			api.ListDestinationsRequest{Name: tc.destination})
		switch {
		case err != nil:
			return poll.Error(err)
		case len(dests.Items) == 0:
			return poll.Continue("destination not yet registered")
		}

		for _, dest := range dests.Items {
			t.Logf("Destination %v (%v) %v", dest.Name, dest.ID, dest.Connection.URL)
		}
		return poll.Success()
	}, poll.WithTimeout(30*time.Second))

	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome) // for windows

	// ssh uses /etc/passwd to find the home directory, not the HOME env
	// variable. So we have to specify the config path explicitly in testing.
	sshConfig := filepath.Join(userHome, ".ssh/config")

	// TODO: what's the right path on darwin?
	infra, err := filepath.Abs("../dist/infra_linux_amd64_v1/infra")
	assert.NilError(t, err)
	t.Setenv("PATH", filepath.Dir(infra)+string(os.PathListSeparator)+os.Getenv("PATH"))
	userKey := "ababababab.000000000000000000000001"

	runStep(t, "login as user", func(t *testing.T) {
		res := icmd.RunCommand(infra,
			"login", "--key="+userKey, "--enable-ssh", infraServerURL,
			"--tls-trusted-cert", infraServerCAFile)
		res.Assert(t, icmd.Success)
	})

	users, err := adminClient.ListUsers(ctx, api.ListUsersRequest{Name: "anyuser@example.com"})
	assert.NilError(t, err)
	assert.Equal(t, len(users.Items), 1)
	user := users.Items[0]

	sshArgs := func(args ...string) []string {
		return append([]string{
			"-p", tc.port,
			"-o", "StrictHostKeyChecking=yes",
			"-o", "PasswordAuthentication=no",
			"-F", sshConfig,
			"127.0.0.1",
		}, args...)
	}

	runStep(t, "fails without grant", func(t *testing.T) {
		res := icmd.RunCommand("ssh", sshArgs("echo", "not ok")...)
		expected := icmd.Expected{
			ExitCode: 255,
			Err:      "Permission denied",
		}
		res.Assert(t, expected)
	})

	var grantIDs []uid.ID
	t.Cleanup(func() {
		for _, id := range grantIDs {
			_ = adminClient.DeleteGrant(ctx, id)
		}
	})

	runStep(t, "succeeds with a grant", func(t *testing.T) {
		resp, err := adminClient.CreateGrant(ctx, &api.GrantRequest{
			UserName:  "anyuser@example.com",
			Resource:  tc.destination,
			Privilege: "connect",
		})
		assert.NilError(t, err)
		grantIDs = append(grantIDs, resp.ID)

		res := icmd.RunCommand("ssh", sshArgs("echo", "ok")...)
		expected := icmd.Expected{Out: "ok"}
		res.Assert(t, expected)
	})

	runStep(t, "fails when grant is removed", func(t *testing.T) {
		err := adminClient.DeleteGrant(ctx, grantIDs[0])
		assert.NilError(t, err)

		res := icmd.RunCommand("ssh", sshArgs("echo", "not ok")...)
		expected := icmd.Expected{
			ExitCode: 255,
			Err:      "Permission denied",
		}
		res.Assert(t, expected)
	})

	group := &api.Group{Name: tc.destination + "-users"}
	t.Cleanup(func() {
		err := adminClient.DeleteGroup(ctx, group.ID)
		var apiError api.Error
		if errors.As(err, &apiError) && apiError.Code == http.StatusNotFound {
			return
		}
		assert.NilError(t, err)
	})

	runStep(t, "succeeds with group grant", func(t *testing.T) {
		groupResp, err := adminClient.CreateGroup(ctx, &api.CreateGroupRequest{Name: group.Name})
		assert.NilError(t, err)
		group.ID = groupResp.ID

		err = adminClient.UpdateUsersInGroup(ctx, &api.UpdateUsersInGroupRequest{
			GroupID:      group.ID,
			UserIDsToAdd: []uid.ID{user.ID},
		})
		assert.NilError(t, err)

		grantResp, err := adminClient.CreateGrant(ctx, &api.GrantRequest{
			GroupName: group.Name,
			Resource:  tc.destination,
			Privilege: "connect",
		})
		assert.NilError(t, err)
		grantIDs = append(grantIDs, grantResp.ID)

		res := icmd.RunCommand("ssh", sshArgs("echo", "ok")...)
		expected := icmd.Expected{Out: "ok"}
		res.Assert(t, expected)
	})

	runStep(t, "fails when group grant is removed", func(t *testing.T) {
		err := adminClient.DeleteGrant(ctx, grantIDs[1])
		assert.NilError(t, err)

		res := icmd.RunCommand("ssh", sshArgs("echo", "not ok")...)
		expected := icmd.Expected{
			ExitCode: 255,
			Err:      "Permission denied",
		}
		res.Assert(t, expected)
	})
}

func httpTransport(t *testing.T, infraServerCAFile string) *http.Transport {
	t.Helper()
	pool := x509.NewCertPool()

	cert, err := os.ReadFile(infraServerCAFile)
	assert.NilError(t, err)

	ok := pool.AppendCertsFromPEM(cert)
	if !ok {
		t.Fatalf("Failed to read trusted certificates for server")
	}

	return &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}
}

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
	}
}
