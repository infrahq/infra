package server

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestLoadConfigEmpty(t *testing.T) {
	s := setupServer(t)

	err := s.loadConfig(BootstrapConfig{})
	assert.NilError(t, err)
}

func TestLoadConfigWithUsers(t *testing.T) {
	s := setupServer(t)

	config := BootstrapConfig{
		Users: []User{
			{
				Name: "bob@example.com",
			},
			{
				Name:     "alice@example.com",
				Password: "password",
			},
			{
				Name:      "sue@example.com",
				AccessKey: "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb",
			},
			{
				Name:      "jim@example.com",
				Password:  "password",
				AccessKey: "bbbbbbbbbb.bbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	user, _, _ := getTestDefaultOrgUserDetails(t, s, "bob@example.com")
	assert.Equal(t, "bob@example.com", user.Name)

	user, creds, _ := getTestDefaultOrgUserDetails(t, s, "alice@example.com")
	assert.Equal(t, "alice@example.com", user.Name)
	err = bcrypt.CompareHashAndPassword(creds.PasswordHash, []byte("password"))
	assert.NilError(t, err)

	user, _, key := getTestDefaultOrgUserDetails(t, s, "sue@example.com")
	assert.Equal(t, "sue@example.com", user.Name)
	assert.Equal(t, key.KeyID, "aaaaaaaaaa")
	chksm := sha256.Sum256([]byte("bbbbbbbbbbbbbbbbbbbbbbbb"))
	assert.Equal(t, bytes.Compare(key.SecretChecksum, chksm[:]), 0) // 0 means the byte slices are equal

	user, creds, key = getTestDefaultOrgUserDetails(t, s, "jim@example.com")
	assert.Equal(t, "jim@example.com", user.Name)
	err = bcrypt.CompareHashAndPassword(creds.PasswordHash, []byte("password"))
	assert.NilError(t, err)
	assert.Equal(t, key.KeyID, "bbbbbbbbbb")
	chksm = sha256.Sum256([]byte("bbbbbbbbbbbbbbbbbbbbbbbb"))
	assert.Equal(t, bytes.Compare(key.SecretChecksum, chksm[:]), 0) // 0 means the byte slices are equal
}

func TestLoadConfigUpdate(t *testing.T) {
	s := setupServer(t)

	config := BootstrapConfig{
		Users: []User{
			{
				Name:      "r2d2@example.com",
				InfraRole: "admin",
			},
			{
				Name:      "c3po@example.com",
				AccessKey: "TllVlekkUz.NFnxSlaPQLosgkNsyzaMttfC",
				InfraRole: "view",
			},
			{
				Name:     "sarah@email.com",
				Password: "supersecret",
			},
		},
	}

	err := s.loadConfig(config)
	assert.NilError(t, err)

	tx := txnForTestCase(t, s.db, s.db.DefaultOrg.ID)
	defaultOrg := s.db.DefaultOrg

	var identities, credentials, accessKeys int64

	grants, err := data.ListGrants(tx, data.ListGrantsOptions{})
	assert.NilError(t, err)
	assert.Assert(t, is.Len(grants, 3)) // 2 from config, 1 internal connector

	privileges := map[string]int{
		"admin":     0,
		"view":      0,
		"connector": 0,
	}

	for _, v := range grants {
		privileges[v.Privilege]++
	}

	assert.Equal(t, privileges["admin"], 1)
	assert.Equal(t, privileges["view"], 1)
	assert.Equal(t, privileges["connector"], 1)

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(4), identities)

	err = tx.QueryRow("SELECT COUNT(*) FROM credentials WHERE organization_id = ?;", defaultOrg.ID).Scan(&credentials)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), credentials) // sarah@example.com

	err = tx.QueryRow("SELECT COUNT(*) FROM access_keys WHERE organization_id = ?;", defaultOrg.ID).Scan(&accessKeys)
	assert.NilError(t, err)
	assert.Equal(t, int64(1), accessKeys) // c3po

	updatedConfig := BootstrapConfig{
		Users: []User{
			{
				Name:      "c3po@example.com",
				InfraRole: "admin",
			},
		},
	}

	err = s.loadConfig(updatedConfig)
	assert.NilError(t, err)

	grants, err = data.ListGrants(tx, data.ListGrantsOptions{})
	assert.NilError(t, err)
	assert.Assert(t, is.Len(grants, 4))

	privileges = map[string]int{
		"admin":     0,
		"view":      0,
		"connector": 0,
	}

	for _, v := range grants {
		privileges[v.Privilege]++
	}

	assert.Equal(t, privileges["admin"], 2)
	assert.Equal(t, privileges["view"], 1)
	assert.Equal(t, privileges["connector"], 1)

	err = tx.QueryRow("SELECT COUNT(*) FROM identities WHERE organization_id = ? AND deleted_at IS null;", defaultOrg.ID).Scan(&identities)
	assert.NilError(t, err)
	assert.Equal(t, int64(4), identities)
}

func TestLoadAccessKey(t *testing.T) {
	s := setupServer(t)

	// access key that we will attempt to assign to multiple users
	testAccessKey := Secret("aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb")

	// create a user and assign them an access key
	bob := &models.Identity{Name: "bob@example.com"}
	err := data.CreateIdentity(s.DB(), bob)
	assert.NilError(t, err)

	err = s.loadAccessKey(s.DB(), bob, testAccessKey)
	assert.NilError(t, err)

	t.Run("access key can be reloaded for the same identity it was issued for", func(t *testing.T) {
		err = s.loadAccessKey(s.DB(), bob, testAccessKey)
		assert.NilError(t, err)
	})

	t.Run("duplicate access key ID is rejected", func(t *testing.T) {
		alice := &models.Identity{Name: "alice@example.com"}
		err = data.CreateIdentity(s.DB(), alice)
		assert.NilError(t, err)

		err = s.loadAccessKey(s.DB(), alice, testAccessKey)
		assert.Error(t, err, "access key assigned to \"alice@example.com\" is already assigned to another user, a user's access key must have a unique ID")
	})
}

// getTestDefaultOrgUserDetails gets the attributes of a user created from a config file
func getTestDefaultOrgUserDetails(t *testing.T, server *Server, name string) (*models.Identity, *models.Credential, *models.AccessKey) {
	t.Helper()
	tx := txnForTestCase(t, server.db, server.db.DefaultOrg.ID)

	user, err := data.GetIdentity(tx, data.GetIdentityOptions{ByName: name})
	assert.NilError(t, err, "user")

	credential, err := data.GetCredentialByUserID(tx, user.ID)
	if !errors.Is(err, internal.ErrNotFound) {
		assert.NilError(t, err, "credentials")
	}

	keys, err := data.ListAccessKeys(tx, data.ListAccessKeyOptions{ByIssuedForID: user.ID})
	if !errors.Is(err, internal.ErrNotFound) {
		assert.NilError(t, err, "access_key")
	}

	// only return the first key
	var accessKey *models.AccessKey
	if len(keys) > 0 {
		accessKey = &keys[0]
	}

	return user, credential, accessKey
}
