package cmd

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server"
)

func TestListCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(dir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	opts := defaultServerOptions(dir)
	opts.BootstrapConfig = server.BootstrapConfig{
		Users: []server.User{
			{Name: "admin", AccessKey: "0000000001.adminadminadminadmin1234", Role: "admin"},
			{Name: "nogrants@example.com", AccessKey: "0000000002.notadminsecretnotadmin02"},
			{Name: "manygrants@example.com", AccessKey: "0000000003.notadminsecretnotadmin03"},
		},
	}
	setupServerOptions(t, &opts)
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx := context.Background()
	runAndWait(ctx, t, srv.Run)

	createGrants(t, srv.DB(),
		api.GrantRequest{UserName: "manygrants@example.com", Resource: "space", Privilege: "explorer"},
		api.GrantRequest{UserName: "manygrants@example.com", Resource: "moon", Privilege: "inhabitant"},
		api.GrantRequest{UserName: "manygrants@example.com", Resource: "infra-this-is-not", Privilege: "view"},
	)

	clientOpts := &APIClientOpts{
		Host:      srv.Addrs.HTTPS.String(),
		AccessKey: "0000000001.adminadminadminadmin1234",
		Transport: httpTransportForHostConfig(&ClientHostConfig{SkipTLSVerify: true}),
	}
	c, err := NewAPIClient(clientOpts)
	assert.NilError(t, err)

	_, err = c.CreateDestination(ctx, &api.CreateDestinationRequest{
		UniqueID: "space",
		Name:     "space",
		Kind:     "kubernetes",
		Connection: api.DestinationConnection{
			URL: "http://localhost:10123/",
			CA:  destinationCA,
		},
	})
	assert.NilError(t, err)

	_, err = c.CreateDestination(ctx, &api.CreateDestinationRequest{
		UniqueID: "moon",
		Name:     "moon",
		Kind:     "ssh",
		Connection: api.DestinationConnection{
			URL: "http://localhost:10124/",
			CA:  destinationCA,
		},
	})
	assert.NilError(t, err)

	_, err = c.CreateDestination(ctx, &api.CreateDestinationRequest{
		UniqueID: "maintain",
		Name:     "infra-this-is-not",
		Connection: api.DestinationConnection{
			URL: "http://localhost:10126/",
			CA:  destinationCA,
		},
	})
	assert.NilError(t, err)

	for _, uniqueID := range []string{"space", "moon", "maintain"} {
		// set client.Headers so each destination becomes connected
		c.Headers = http.Header{
			"Infra-Destination": {uniqueID},
		}

		_, err := c.ListGrants(ctx, api.ListGrantsRequest{})
		assert.NilError(t, err)
	}

	// reset client.Headers
	c.Headers = http.Header{}

	users, err := c.ListUsers(ctx, api.ListUsersRequest{})
	assert.NilError(t, err)

	userMap := usersToMap(users.Items)

	t.Run("with no grants", func(t *testing.T) {
		user := userMap["nogrants@example.com"]
		err := writeConfig(&ClientConfig{
			ClientConfigVersion: clientConfigVersion,
			Hosts: []ClientHostConfig{
				{
					UserID:        user.ID,
					Name:          user.Name,
					Host:          srv.Addrs.HTTPS.String(),
					AccessKey:     "0000000002.notadminsecretnotadmin02",
					SkipTLSVerify: true,
					Expires:       api.Time(time.Now().Add(5 * time.Second)),
					Current:       true,
				},
			},
		})
		assert.NilError(t, err)

		ctx, bufs := PatchCLI(ctx)
		err = Run(ctx, "list")
		assert.NilError(t, err)

		expected := "You have not been granted access to any active destinations\n"
		assert.Equal(t, bufs.Stdout.String(), expected)
	})

	t.Run("with many grants", func(t *testing.T) {
		user := userMap["manygrants@example.com"]
		err := writeConfig(&ClientConfig{
			ClientConfigVersion: clientConfigVersion,
			Hosts: []ClientHostConfig{
				{
					UserID:        user.ID,
					Name:          user.Name,
					Host:          srv.Addrs.HTTPS.String(),
					AccessKey:     "0000000003.notadminsecretnotadmin03",
					SkipTLSVerify: true,
					Expires:       api.Time(time.Now().Add(5 * time.Second)),
					Current:       true,
				},
			},
		})
		assert.NilError(t, err)

		ctx, bufs := PatchCLI(ctx)
		err = Run(ctx, "list")
		assert.NilError(t, err)
		golden.Assert(t, bufs.Stdout.String(), t.Name())
	})
}

func usersToMap(users []api.User) map[string]api.User {
	result := make(map[string]api.User)
	for _, u := range users {
		result[u.Name] = u
	}
	return result
}
