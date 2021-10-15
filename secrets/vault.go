package secrets

import (
	"encoding/base64"
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

var DefaultVaultAlgorithm = "aes256-gcm96"

type VaultSecretProvider struct {
	TransitMount string `yaml:"transit_mount"` // mounting point. defaults to /transit
	SecretMount  string `yaml:"secret_mount"`  // mounting point. defaults to /secret
	Token        string `yaml:"token"`         // vault token... should authenticate as machine to vault instead?
	Namespace    string `yaml:"namespace"`

	client *vault.Client
}

func NewVaultSecretProvider(address, token, namespace string) (*VaultSecretProvider, error) {
	c, err := vault.NewClient(&vault.Config{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	c.SetToken(token)

	if len(namespace) > 0 {
		c.SetNamespace(namespace)
	}

	v := &VaultSecretProvider{
		TransitMount: "/transit",
		SecretMount:  "/secret",
		Token:        token,
		Namespace:    namespace,
		client:       c,
	}

	return v, nil
}

func (v *VaultSecretProvider) GetSecret(name string) ([]byte, error) {
	path := fmt.Sprintf("%s/data/%s", v.SecretMount, name)

	sec, err := v.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}

	if data, ok := sec.Data["data"].(map[string]interface{})["data"].(string); ok {
		return []byte(data), nil
	}

	return nil, nil
}

func (v *VaultSecretProvider) SetSecret(name string, secret []byte) error {
	path := fmt.Sprintf("%s/data/%s", v.SecretMount, name)
	_, err := v.client.Logical().Write(path, map[string]interface{}{
		"data": map[string]interface{}{
			"data": string(secret),
		},
	})

	return err
}

func (v *VaultSecretProvider) GenerateDataKey(name, rootKeyID string) (*SymmetricKey, error) {
	if rootKeyID == "" {
		rootKeyID = name + "_root"
		if err := v.generateRootKey(rootKeyID); err != nil {
			return nil, err
		}
	}
	// generate a new data key
	dataKey, err := cryptoRandRead(32) // 256 bit
	if err != nil {
		return nil, fmt.Errorf("generating data key: %w", err)
	}

	// encrypt the data key
	encrypted, err := v.RemoteEncrypt(rootKeyID, dataKey)
	if err != nil {
		return nil, fmt.Errorf("remote encrypt: %w", err)
	}

	return &SymmetricKey{
		unencrypted: dataKey,
		Encrypted:   encrypted,
		Algorithm:   DefaultVaultAlgorithm,
		RootKeyID:   rootKeyID,
	}, nil
}

func (v *VaultSecretProvider) generateRootKey(name string) error {
	path := fmt.Sprintf("%s/keys/%s", v.TransitMount, name)

	_, err := v.client.Logical().Write(path, map[string]interface{}{
		"convergent_encryption":  false,
		"derived":                false,
		"exportable":             false,
		"allow_plaintext_backup": false,
		"type":                   DefaultVaultAlgorithm,
	})

	return err
}

func (v *VaultSecretProvider) DecryptDataKey(rootKeyID string, keyData []byte) (*SymmetricKey, error) {
	plain, err := v.RemoteDecrypt(rootKeyID, keyData)
	if err != nil {
		return nil, err
	}

	return &SymmetricKey{
		unencrypted: plain,
		Encrypted:   keyData,
		Algorithm:   DefaultVaultAlgorithm,
		RootKeyID:   rootKeyID,
	}, nil
}

func (v *VaultSecretProvider) RemoteEncrypt(keyID string, plain []byte) (encrypted []byte, err error) {
	bPlain := base64.StdEncoding.EncodeToString(plain)

	sec, err := v.client.Logical().Write("/transit/encrypt/"+keyID, map[string]interface{}{
		"plaintext": bPlain,
	})
	if err != nil {
		return nil, err
	}

	if data, ok := sec.Data["ciphertext"].(string); ok {
		return []byte(data), nil
	}

	return nil, nil
}

func (v *VaultSecretProvider) RemoteDecrypt(keyID string, encrypted []byte) (plain []byte, err error) {
	sec, err := v.client.Logical().Write("/transit/decrypt/"+keyID, map[string]interface{}{
		"ciphertext": string(encrypted),
	})
	if err != nil {
		return nil, err
	}

	if data, ok := sec.Data["plaintext"].(string); ok {
		d, err := base64.StdEncoding.DecodeString(data)
		return d, err
	}

	return nil, nil
}
