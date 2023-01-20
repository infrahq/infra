package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type keysConfig struct {
	Keys []localPublicKey
}

type localPublicKey struct {
	// Server is the host:port of the Infra API server where this key was
	// uploaded. This value matches the Host value stored in ClientConfig.
	Server string
	// OrganizationID is the organization ID where this key was uploaded.
	OrganizationID string
	// UserID is the infra user ID of the user who uploaded this public key.
	UserID string
	// PublicKeyID is the Infra ID of the UserPublicKey. It's also used as the
	// name of the local file which should store the private and public key pair.
	PublicKeyID string
}

// readKeysConfig reads ~/.ssh/infra/keys.json and returns the contents.
func readKeysConfig(infraSSHDir string) (*keysConfig, error) {
	filename := filepath.Join(infraSSHDir, "keys.json")
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close() // read-only file

	result := &keysConfig{}
	err = json.NewDecoder(fh).Decode(result)
	return result, err
}

func writeKeysConfig(infraSSHDir string, cfg *keysConfig) error {
	filename := filepath.Join(infraSSHDir, "keys.json")
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(fh).Encode(cfg); err != nil {
		_ = fh.Close() // prefer the write error over the close error
		return err
	}
	return fh.Close()
}

// publicKeyMatches matches localKey against hostCfg and OrgID. Returns true
// when the localKey matches the host, user and org.
func publicKeyMatches(localKey localPublicKey, hostCfg *ClientHostConfig, orgID uid.ID) bool {
	return localKey.Server == hostCfg.Host &&
		localKey.UserID == hostCfg.UserID.String() &&
		localKey.OrganizationID == orgID.String()
}

func userPublicKeyContains(keys []api.UserPublicKey, id string) bool {
	for _, key := range keys {
		if key.ID.String() == id {
			return true
		}
	}
	return false
}
