package secrets

import (
	"encoding/base64"
	"fmt"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

var DefaultVaultAlgorithm = "aes256-gcm96"

// ensure these interfaces are implemented properly
var (
	_ SecretSymmetricKeyProvider = &VaultSecretProvider{}
	_ SecretStorage              = &VaultSecretProvider{}
)

type VaultSecretProvider struct {
	VaultConfig
	client *vault.Client
}

type VaultConfig struct {
	TransitMount string `yaml:"transit_mount"` // mounting point. defaults to /transit
	SecretMount  string `yaml:"secret_mount"`  // mounting point. defaults to /secret
	Token        string `yaml:"token"`         // vault token... should authenticate as machine to vault instead?
	Namespace    string `yaml:"namespace"`
	Address      string `yaml:"address"`
}

func NewVaultConfig() VaultConfig {
	return VaultConfig{
		TransitMount: "/transit",
		SecretMount:  "/secret",
		Address:      "https://vault",
	}
}

func NewVaultSecretProviderFromConfig(cfg VaultConfig) (*VaultSecretProvider, error) {
	c, err := vault.NewClient(&vault.Config{
		Address: cfg.Address,
	})
	if err != nil {
		return nil, err
	}

	c.SetToken(cfg.Token)

	if len(cfg.Namespace) > 0 {
		c.SetNamespace(cfg.Namespace)
	}

	v := &VaultSecretProvider{
		VaultConfig: cfg,
		client:      c,
	}

	return v, nil
}

func NewVaultSecretProvider(address, token, namespace string) (*VaultSecretProvider, error) {
	return NewVaultSecretProviderFromConfig(VaultConfig{
		Address:   address,
		Token:     token,
		Namespace: namespace,
	})
}

func (v *VaultSecretProvider) GetSecret(name string) ([]byte, error) {
	name = nameEscape(name)
	path := fmt.Sprintf("%s/data/%s", v.SecretMount, name)

	sec, err := v.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}

	if sec == nil || sec.Data == nil {
		return nil, nil
	}

	if _, ok := sec.Data["data"]; !ok {
		return nil, nil
	}

	data, ok := sec.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("vault: secret data is unexpected not stored in a map")
	}

	if data, ok := data["data"].(string); ok {
		return []byte(data), nil
	}

	return nil, fmt.Errorf("vault: secret data is not a string")
}

func (v *VaultSecretProvider) SetSecret(name string, secret []byte) error {
	name = nameEscape(name)
	path := fmt.Sprintf("%s/data/%s", v.SecretMount, name)
	_, err := v.client.Logical().Write(path, map[string]interface{}{
		"data": map[string]interface{}{
			"data": string(secret),
		},
	})

	return err
}

func (v *VaultSecretProvider) GenerateDataKey(name, rootKeyID string) (*SymmetricKey, error) {
	name = nameEscape(name)
	if rootKeyID == "" {
		rootKeyID = name + "_root"
		if err := v.generateRootKey(rootKeyID); err != nil {
			return nil, err
		}
	}
	// generate a new data key
	dataKey, err := cryptoRandRead(32) // 256 bit
	if err != nil {
		return nil, fmt.Errorf("vault: generating data key: %w", err)
	}

	// encrypt the data key
	encrypted, err := v.RemoteEncrypt(rootKeyID, dataKey)
	if err != nil {
		return nil, fmt.Errorf("vault: remote encrypt: %w", err)
	}

	return &SymmetricKey{
		unencrypted: dataKey,
		Encrypted:   encrypted,
		Algorithm:   DefaultVaultAlgorithm,
		RootKeyID:   rootKeyID,
	}, nil
}

func (v *VaultSecretProvider) generateRootKey(name string) error {
	name = nameEscape(name)
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

func nameEscape(name string) string {
	rpl := strings.NewReplacer(
		"/", "_",
		":", "_",
	)

	return rpl.Replace(name)
}
