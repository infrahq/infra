package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/connector"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestSSHDAuthKeysCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("INFRA_LOG_LEVEL", "debug")

	etcPasswdFilename = "testdata/sshd-auth-keys/etcpasswd" //nolint:gosec
	t.Cleanup(func() {
		etcPasswdFilename = "/etc/passwd"
	})

	opts := defaultServerOptions(home)
	opts.BootstrapConfig = server.BootstrapConfig{
		Users: []server.User{
			{
				Name:      "admin@example.com",
				AccessKey: "0000000001.adminadminadminadmin1234",
				Role:      "admin",
			},
			{Name: "connector", AccessKey: "0000000003.connectorsecretconnector"},
			{Name: "anyuser@example.com", AccessKey: "0000000002.notadminsecretnotadmin02"},
			{Name: "otheruser@example.com"},
			{Name: "nogrant@example.com"},
		},
	}
	setupServerOptions(t, &opts)
	srv, err := server.New(opts)
	assert.NilError(t, err)

	ctx := context.Background()
	runAndWait(ctx, t, srv.Run)

	createGrants(t, srv.DB(),
		api.GrantRequest{UserName: "anyuser@example.com", Resource: "prodhost", Privilege: "connect"},
		api.GrantRequest{UserName: "otheruser@example.com", Resource: "prodhost", Privilege: "connect"},
		api.GrantRequest{UserName: "nogrant@example.com", Resource: "otherhost", Privilege: "connect"},
	)

	pubKey := `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCxB1cFqocoje6xEGj3UOlNMo4b51ff7F7V4FzVsVyGk2iDYy/ZwuFfdAnKVQETCY/jwpKw6UQp7Sg1E5R9YljyCRjSGJXY1Tv07HJsYF8z4vVXpV15Sp4md9ExB0EGkdtagb10pX3lnj5vZSur6NvdsXWYh8ikZZydB3KKCV3ylgb2OOzGpSHD9MEc4b1LUyFAqB7zZeiccDYgIqwZ3spuX7Kt3vrC46H1Fv9yWjnZ4S1xJYHVgDwBTJE3rszVzX5ZHCbdvWMKBbvnzZlh8GBwxgoH4MEPnhTZSCk26BtFjSGyVG3CXsI0o4uJERw+oqSG/A45LN+qa0e+0O54VylIgploM0+inWDL7tInUjkFIFd6qhqxELGVpE8BOrw8ucW8xfmWyCISI9W9Z482HK2/SCuFWCJaPxHEOgLjYwB4aTEMbLSewRRRBUC1J4hmIp23Hu2yYuE7kC8w7zWptw43qLvWy4SAdCZFEpR+hSRD77nnsgabz4HGGnECAFwRXA0=`
	addUserPublicKeyByEmail(t, srv.DB(), "anyuser@example.com", pubKey)

	pubKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDZmP6rrpvBi5E4TftsQUSbaTusQCnxNSsWHWpcdiFyUu8CBJ9Qmyljgt3NPJjEbAB4Pn1gEPbEjrEpXGcRssP3hhrD/elMJNiTGNuv2GB6sHy/pY4Hpdv/vpwnypbsMfMvP6caznvHHtkD92XfxPRtChODVIu95CkWEG+IJGbjZZ4Q6ff3EJ6BMOFBdHLKjHJ5OcAL10tvrPVr+i2OqhvT4inL59ZqLaxOaJwJIN8Wicy6MFeQfKxE6D87GvjRl+2HiIt44EMi2c5+6uNMahZv7oz60A+ej5ba8mEgo4nbDA2YKJRrW6fD+knYzWLOCV0a7jTwAnGT+v+pEnmHS3ccznoLqKIlo6hWMjW1LTCwr+Eus+nYJoBcaPRPdL8PE7NgosdwfrGCT7EhWkATzout543gXlYAMQROb9xeLFYRkfOAZuoVmkEzOwc0K5O0zUQsGr7bIhRiGcIoEi25WabQIQCkR5Err+Ov+AsmQ6vuQ2ONVMC4c00ColkqGdahpLk=`
	addUserPublicKeyByEmail(t, srv.DB(), "otheruser@example.com", pubKey)

	pubKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDPkW3mIACvMmXqbeGF/U2MY8jbQ5NT24tRL0cl+32vRMmIDGcEyLkWh98D9qJlwCIZ8vJahAI3sqYJRoIHkiaRTslWwAZWNnTJ3TzeKUn/g0xutASD4znmQhNk3OuKPyuDKRxvsOuBVzuKiNNeUWVf5v/4gPrmBffS19cPPlHG+TwHNzTvyvbLcZu+xE18x8eCM4uRam0wa4RfHrMtaqPb/kFGz7skXv0/JFCXKrc//dMKHbr/brjj7fKYFYbMG7k15LewfZ/fLqsbJsvuP8OTIE7195fKhL1Gln8AKOM1E0CLX9nxK7qx4MlrDgEJBbqikWb2kVKmpxwcA7UcoUbwKZb4/QrOUDy22aHnIErIl2is9IP8RfBdKgzmgT1QmVPcGHI4gBAPb279zw58nAVp58gzHvK/oTDlAD2zq87i/PeDSzdoVZe0zliKOXAVzLQGI+9vsZ+6URHBe6J+Tj+PxOD5sWduhepOa/UKF96+CeEg/oso4UHR83z5zR38idc=`
	addUserPublicKeyByEmail(t, srv.DB(), "nogrant@example.com", pubKey)

	connectorOpts := connector.Options{
		Name: "prodhost",
		Server: connector.ServerOptions{
			URL:                urlFromAddr(t, srv.Addrs.HTTPS),
			AccessKey:          "0000000003.connectorsecretconnector",
			TrustedCertificate: opts.TLS.Certificate,
		},
	}
	raw, err := json.Marshal(connectorOpts)
	assert.NilError(t, err)
	connectorConfig := filepath.Join(home, "connector.yaml")
	err = os.WriteFile(connectorConfig, raw, 0600)
	assert.NilError(t, err)

	type testCase struct {
		name                 string
		username             string
		publicKeyFingerprint string
		expected             string
	}

	run := func(t *testing.T, tc testCase) {
		err := Run(ctx, "sshd", "auth-keys",
			"--config-file", connectorConfig,
			tc.username, tc.publicKeyFingerprint)
		if tc.expected == "" {
			assert.NilError(t, err)
			return
		}

		assert.ErrorContains(t, err, tc.expected)
	}

	testCases := []testCase{
		{
			name:                 "success",
			username:             "anyuser",
			publicKeyFingerprint: "SHA256:Ek9z81rm9t5KhBmlUZLbbpcYogOU2JR4nBaoHB70bmY",
		},
		{
			name:                 "auth failed unknown public key",
			username:             "anyuser",
			publicKeyFingerprint: "SHA256:Ek9z81rm9t5KhBmlUZLaaaaaaaaaaaaaaaaoHB70bmY",
			expected:             "wrong number of users found 0",
		},
		{
			name:                 "auth failed wrong user for pubkey",
			username:             "other",
			publicKeyFingerprint: "SHA256:Ek9z81rm9t5KhBmlUZLbbpcYogOU2JR4nBaoHB70bmY",
			expected:             "public key is for a different user",
		},
		{
			name:                 "auth failed user not managed by infra",
			username:             "otheruser",
			publicKeyFingerprint: "SHA256:KbbiBxEGEiy28rL4AKx06oAmG4V3fmLMTzgk6zOxPBE",
			expected:             "user is not managed by infra",
		},
		{
			name:                 "auth failed no grant for destination",
			username:             "nogrant",
			publicKeyFingerprint: "SHA256:dwF3R8L454kABUAJc+ZdJeaV2xbcXVJfb81tuv/1KLo",
			expected:             "has not been granted access to this destination",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func addUserPublicKeyByEmail(t *testing.T, db data.WriteTxn, email string, pubKey string) {
	t.Helper()
	user, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: email})
	assert.NilError(t, err, email)

	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	assert.NilError(t, err)

	userPublicKey := &models.UserPublicKey{
		UserID:      user.ID,
		PublicKey:   base64.StdEncoding.EncodeToString(key.Marshal()),
		KeyType:     key.Type(),
		Fingerprint: ssh.FingerprintSHA256(key),
	}
	err = data.AddUserPublicKey(db, userPublicKey)
	assert.NilError(t, err)
}
