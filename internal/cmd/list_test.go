package cmd

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/uid"
)

func TestListCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	// k8s.io/tools/clientcmd reads HOME at import time, so this must be patched too
	kubeConfigPath := filepath.Join(dir, "kube.config")
	t.Setenv("KUBECONFIG", kubeConfigPath)

	opts := defaultServerOptions(dir)
	opts.Config = server.Config{
		Users: []server.User{
			{Name: "admin", AccessKey: "0000000001.adminadminadminadmin1234"},
			{Name: "nogrants@example.com", AccessKey: "0000000002.notadminsecretnotadmin02"},
			{Name: "manygrants@example.com", AccessKey: "0000000003.notadminsecretnotadmin03"},
		},
		Grants: []server.Grant{
			{User: "admin", Resource: "infra", Role: "admin"},
			{User: "manygrants@example.com", Resource: "space", Role: "explorer"},
			{User: "manygrants@example.com", Resource: "moon", Role: "inhabitant"},
		},
	}
	opts.Addr = server.ListenerOptions{HTTPS: "127.0.0.1:0", HTTP: "127.0.0.1:0"}
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	setupCertManager(t, opts.TLSCache, srv.Addrs.HTTPS.String())
	go func() {
		assert.Check(t, srv.Run(ctx))
	}()

	c, err := apiClient(srv.Addrs.HTTPS.String(), "0000000001.adminadminadminadmin1234", true)
	assert.NilError(t, err)

	_, err = c.CreateDestination(&api.CreateDestinationRequest{
		UniqueID: "space",
		Name:     "space",
		Connection: api.DestinationConnection{
			URL: "http://localhost:10123/",
			CA:  destinationCA,
		},
	})
	assert.NilError(t, err)

	_, err = c.CreateDestination(&api.CreateDestinationRequest{
		UniqueID: "moon",
		Name:     "moon",
		Connection: api.DestinationConnection{
			URL: "http://localhost:10124/",
			CA:  destinationCA,
		},
	})
	assert.NilError(t, err)

	users, err := c.ListUsers(api.ListUsersRequest{})
	assert.NilError(t, err)

	userMap := usersToMap(users.Items)

	t.Run("with no grants", func(t *testing.T) {
		user := userMap["nogrants@example.com"]
		err := writeConfig(&ClientConfig{
			Version: "0.3",
			Hosts: []ClientHostConfig{
				{
					PolymorphicID: uid.NewIdentityPolymorphicID(user.ID),
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
			Version: "0.3",
			Hosts: []ClientHostConfig{
				{
					PolymorphicID: uid.NewIdentityPolymorphicID(user.ID),
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
