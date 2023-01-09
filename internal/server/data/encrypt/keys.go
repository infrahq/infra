package encrypt

import (
	"fmt"
	"os"
)

var (
	keyBlockSizeInBits  = 256
	keyBlockSizeInBytes = keyBlockSizeInBits / 8
	algorithmAESGCM     = "aesgcm"
)

func CreateRootKey(filename string) error {
	rootKey, err := cryptoRandRead(keyBlockSizeInBytes)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, rootKey, 0o600)
}

func CreateDataKey(rootKeyPath string) (*SymmetricKey, error) {
	rootKey, err := os.ReadFile(rootKeyPath)
	if err != nil {
		return nil, err
	}

	fullRootKey := &SymmetricKey{
		unencrypted: rootKey,
		Algorithm:   algorithmAESGCM,
	}

	dataKey, err := cryptoRandRead(keyBlockSizeInBytes)
	if err != nil {
		return nil, err
	}

	encDataKey, err := Seal(fullRootKey, dataKey)
	if err != nil {
		return nil, fmt.Errorf("sealing: %w", err)
	}

	return &SymmetricKey{
		unencrypted: dataKey,
		Encrypted:   encDataKey,
		Algorithm:   algorithmAESGCM,
		RootKeyID:   rootKeyPath,
	}, nil
}

func DecryptDataKey(rootKeyPath string, keyData []byte) (*SymmetricKey, error) {
	rootKey, err := os.ReadFile(rootKeyPath)
	if err != nil {
		return nil, fmt.Errorf("getting root key: %w", err)
	}

	fullRootKey := &SymmetricKey{
		unencrypted: rootKey,
		Algorithm:   algorithmAESGCM,
	}

	unsealed, err := Unseal(fullRootKey, keyData)
	if err != nil {
		return nil, fmt.Errorf("unsealing: %w", err)
	}

	return &SymmetricKey{
		unencrypted: unsealed,
		Encrypted:   keyData,
		Algorithm:   algorithmAESGCM,
		RootKeyID:   rootKeyPath,
	}, nil
}
