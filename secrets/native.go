package secrets

import "fmt"

var (
	infraKeyNamespace   = "infra-x"                         // k8s doesn't like names that start or end with _
	infraRootKeyID      = infraKeyNamespace + "/__root_key" // k8s requires one slash in the name
	keyBlockSizeInBits  = 256
	keyBlockSizeInBytes = keyBlockSizeInBits / 8
)

var (
	AlgorithmAESGCM = "aesgcm"
)

func NewNativeSecretProvider(storage SecretStorage) *NativeSecretProvider {
	return &NativeSecretProvider{
		SecretStorage: storage,
	}
}

type NativeSecretProvider struct {
	SecretStorage SecretStorage
}

func (n *NativeSecretProvider) GenerateDataKey(rootKeyID string) (*SymmetricKey, error) {
	if rootKeyID == "" {
		rootKeyID = infraRootKeyID
	}

	rootKey, err := n.SecretStorage.GetSecret(rootKeyID)
	if err != nil {
		return nil, fmt.Errorf("getting root key: %w", err)
	}

	if len(rootKey) == 0 {
		// generate root key
		rootKey, err = cryptoRandRead(keyBlockSizeInBytes)
		if err != nil {
			return nil, err
		}

		if err = n.SecretStorage.SetSecret(rootKeyID, rootKey); err != nil {
			return nil, fmt.Errorf("saving root key: %w", err)
		}
	}

	fullRootKey := &SymmetricKey{
		unencrypted: rootKey,
		Algorithm:   AlgorithmAESGCM,
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
		Algorithm:   AlgorithmAESGCM,
		RootKeyID:   rootKeyID,
	}, nil
}

func (n *NativeSecretProvider) DecryptDataKey(rootKeyID string, keyData []byte) (*SymmetricKey, error) {
	if rootKeyID == "" {
		rootKeyID = infraRootKeyID
	}

	rootKey, err := n.SecretStorage.GetSecret(rootKeyID)
	if err != nil {
		return nil, fmt.Errorf("getting root key: %w", err)
	}

	fullRootKey := &SymmetricKey{
		unencrypted: rootKey,
		Algorithm:   AlgorithmAESGCM,
	}

	unsealed, err := Unseal(fullRootKey, keyData)
	if err != nil {
		return nil, fmt.Errorf("unsealing: %w", err)
	}

	return &SymmetricKey{
		unencrypted: unsealed,
		Encrypted:   keyData,
		Algorithm:   AlgorithmAESGCM,
		RootKeyID:   rootKeyID,
	}, nil
}
