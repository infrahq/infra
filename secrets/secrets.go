package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// SecretStorage is implemented by a provider if the provider gives a mechanism for storing arbitrary secrets.
type SecretStorage interface {
	// Use secrets when you don't want to store the underlying data, eg secret tokens
	SetSecret(name string, secret []byte) error
	GetSecret(name string) (secret []byte, err error)
}

// SecretProvider is implemented by a provider that provides encryption-as-a-service.
// Its use is opinionated about the provider in the following ways:
// - A root key will be created or referenced and never leaves the provider
// - the root key will be used to encrypt a "data key"
// - the data key is given to the client (us) for encrypting data
// - the client shall store only the encrypted data key
// - the client shall remove the plaintext data key from memory as soon as it is no longer needed
// - the client will request the data key be decrypted by the provider if it is needed subsequently.
// In this way the encryption-as-a-service provider scales to unlimited data sizes without needing to transfer the data to the remote service for symmetric encryption/decryption.
// To rotate root keys, generate new ones periodically and reencrypt data you touch with the new root. This can either be done all at once or gradually over time. Old root keys are out of circulation when no data exists that points to them.
type SecretProvider interface {
	// GenerateDataKey makes a data key from a root key id: if "", a root key is created. It is okay to generate many data keys.
	GenerateDataKey(name, rootKeyID string) (*SymmetricKey, error)
	// DecryptDataKey decrypts the encrypted data key on the provider given a root key id
	DecryptDataKey(rootKeyID string, keyData []byte) (*SymmetricKey, error)
}

type SymmetricKey struct {
	unencrypted []byte `json:"-"`    // the unencrypted data key. Retrieved with DecryptDataKey or set by GenerateDataKey. This field *MUST NOT* be persisted.
	Encrypted   []byte `json:"key"`  // the encrypted data key. To be stored by caller.
	Algorithm   string `json:"alg"`  // Algorithm key used for encryption. To be stored by caller.
	RootKeyID   string `json:"rkid"` // ID of the root key used to encrypt the data key on the provider. To be stored by caller.
}

type encryptedPayload struct {
	Ciphertext []byte `json:"d"` // base64 encoded
	Algorithm  string `json:"a"` // name of the algorithm used to encrypt the Ciphertext
	Key        []byte `json:"k"` // encrypted key data
	RootKeyID  string `json:"i"` // id of the root key the Key field is encrypted with
	Nonce      []byte `json:"n"` // must be crypto random unique every time, size = block size
}

// cryptoRandRead is a safe read from crypto/rand, checking errors and number of bytes read, erroring if we don't get enough
func cryptoRandRead(length int) ([]byte, error) {
	b := make([]byte, length)

	i, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("crypto/rand read: %w", err)
	}

	if i != length {
		return nil, fmt.Errorf("could not read %d random characters from crypto/rand, only got %d", length, i)
	}

	return b, nil
}

// Seal encrypts plaintext with a decrypted data key
func Seal(key *SymmetricKey, plain []byte) ([]byte, error) {
	if len(key.unencrypted) == 0 {
		return nil, errors.New("missing key")
	}

	if len(key.unencrypted) != 32 {
		return nil, errors.New("expected 256 bit key size")
	}

	blk, err := aes.NewCipher(key.unencrypted)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}

	nonce, err := cryptoRandRead(aesgcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	encrypted := aesgcm.Seal(nil, nonce, plain, nil)

	payload := encryptedPayload{
		Ciphertext: encrypted,
		Algorithm:  key.Algorithm,
		Key:        key.Encrypted,
		RootKeyID:  key.RootKeyID,
		Nonce:      nonce,
	}

	jsonPayload, err := json.Marshal(&payload)
	if err != nil {
		return nil, err
	}

	encoded := make([]byte, base64.RawStdEncoding.EncodedLen(len(jsonPayload)))
	base64.RawStdEncoding.Encode(encoded, jsonPayload)

	return encoded, nil
}

// Unseal decrypts ciphertext with a decrypted data key
func Unseal(key *SymmetricKey, encoded []byte) ([]byte, error) {
	if len(key.unencrypted) == 0 {
		return nil, errors.New("missing key")
	}

	jsonPayload := make([]byte, base64.RawStdEncoding.DecodedLen(len(encoded)))

	_, err := base64.RawStdEncoding.Decode(jsonPayload, encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	payload := &encryptedPayload{}
	if err := json.Unmarshal(jsonPayload, payload); err != nil {
		return nil, fmt.Errorf("unmarshalling: %w", err)
	}

	blk, err := aes.NewCipher(key.unencrypted)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, payload.Nonce, payload.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("opening seal: %w", err)
	}

	return plaintext, nil
}
