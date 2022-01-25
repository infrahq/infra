package secrets

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrNotFound = fmt.Errorf("secret not found")

// SecretStorage is implemented by a provider if the provider gives a mechanism for storing arbitrary secrets.
type SecretStorage interface {
	// Use secrets when you don't want to store the underlying data, eg secret tokens
	SetSecret(name string, secret []byte) error
	GetSecret(name string) (secret []byte, err error)
}

var SecretStorageProviderKinds = []string{
	"vault",
	"awsssm",
	"awssecretsmanager",
	"kubernetes",
	"env",
	"file",
	"plaintext",
}

// SymmetricKeyProvider is implemented by a provider that provides encryption-as-a-service.
// Its use is opinionated about the provider in the following ways:
// - A root key will be created or referenced and never leaves the provider
// - the root key will be used to encrypt a "data key"
// - the data key is given to the client (us) for encrypting data
// - the client shall store only the encrypted data key
// - the client shall remove the plaintext data key from memory as soon as it is no longer needed
// - the client will request the data key be decrypted by the provider if it is needed subsequently.
// In this way the encryption-as-a-service provider scales to unlimited data sizes without needing to transfer the data to the remote service for symmetric encryption/decryption.
// To rotate root keys, generate new ones periodically and reencrypt data you touch with the new root. This can either be done all at once or gradually over time. Old root keys are out of circulation when no data exists that points to them.
type SymmetricKeyProvider interface {
	// GenerateDataKey makes a data key from a root key id: if "", a root key is created. It is okay to generate many data keys.
	GenerateDataKey(rootKeyID string) (*SymmetricKey, error)
	// DecryptDataKey decrypts the encrypted data key on the provider given a root key id
	DecryptDataKey(rootKeyID string, keyData []byte) (*SymmetricKey, error)
}

var SymmetricKeyProviderKinds = []string{
	"vault",
	"awskms",
	"native",
}

type SymmetricKey struct {
	unencrypted []byte `json:"-"`    // the unencrypted data key. Retrieved with DecryptDataKey or set by GenerateDataKey. This field *MUST NOT* be persisted.
	Encrypted   []byte `json:"key"`  // the encrypted data key. To be stored by caller.
	Algorithm   string `json:"alg"`  // Algorithm key used for encryption. To be stored by caller.
	RootKeyID   string `json:"rkid"` // ID of the root key used to encrypt the data key on the provider. To be stored by caller.
}

type encryptedPayload struct {
	Ciphertext []byte // base64 encoded
	Algorithm  string // name of the algorithm used to encrypt the Ciphertext
	KeyID      []byte // A unique identifier for the key used. Could be checksum or a version number
	RootKeyID  string // id of the root key the Key field is encrypted with
	Nonce      []byte // must be crypto-random unique every time, size = block size
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

// SealRaw encrypts plaintext with a decrypted data key and returns it in a raw binary format
func SealRaw(key *SymmetricKey, plain []byte) ([]byte, error) {
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

	// TODO: nonce could be incremented after a single random generation on init
	nonce, err := cryptoRandRead(aesgcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	encrypted := aesgcm.Seal(nil, nonce, plain, nil)

	keyID, err := checksum(key.Encrypted)
	if err != nil {
		return nil, err
	}

	payload := encryptedPayload{
		KeyID:      keyID,
		Ciphertext: encrypted,
		Algorithm:  key.Algorithm,
		RootKeyID:  key.RootKeyID,
		Nonce:      nonce,
	}

	marshalledPayload, err := marshalPayload(&payload)
	if err != nil {
		return nil, err
	}

	return marshalledPayload, nil
}

// Seal encrypts plaintext with a decrypted data key and returns it in base64
func Seal(key *SymmetricKey, plain []byte) ([]byte, error) {
	marshalled, err := SealRaw(key, plain)
	if err != nil {
		return nil, err
	}

	encoded := make([]byte, base64.RawStdEncoding.EncodedLen(len(marshalled)))
	base64.RawStdEncoding.Encode(encoded, marshalled)

	return encoded, nil
}

// UnsealRaw decrypts ciphertext with a decrypted data key and returns a raw binary format
func UnsealRaw(key *SymmetricKey, encrypted []byte) ([]byte, error) {
	if len(key.unencrypted) == 0 {
		return nil, errors.New("missing key")
	}

	payload := &encryptedPayload{}
	if err := unmarshalPayload(encrypted, payload); err != nil {
		return nil, fmt.Errorf("unmarshalling: %w", err)
	}

	ck, err := checksum(key.Encrypted)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(ck, payload.KeyID) {
		return nil, fmt.Errorf("supplied key cannot decrypt this message; wrong key was used")
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

// Unseal decrypts base64-encoded ciphertext with a decrypted data key
func Unseal(key *SymmetricKey, encoded []byte) ([]byte, error) {
	encryptedPayload := make([]byte, base64.RawStdEncoding.DecodedLen(len(encoded)))

	_, err := base64.RawStdEncoding.Decode(encryptedPayload, encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	return UnsealRaw(key, encryptedPayload)
}

func checksum(b []byte) ([]byte, error) {
	keyIDChecksum := sha256.New()

	ln, err := keyIDChecksum.Write(b)
	if err != nil {
		return nil, fmt.Errorf("creating checksum: %w", err)
	}

	if ln != len(b) {
		return nil, fmt.Errorf("checksum write len")
	}

	return keyIDChecksum.Sum(nil)[:4], nil
}

func unmarshalPayload(mp []byte, p *encryptedPayload) error {
	b := bytes.NewBuffer(mp)

	var ln int32
	if err := binary.Read(b, binary.BigEndian, &ln); err != nil {
		return err
	}

	p.Ciphertext = make([]byte, ln)
	if err := binary.Read(b, binary.BigEndian, &p.Ciphertext); err != nil {
		return err
	}

	var length uint8
	if err := binary.Read(b, binary.BigEndian, &length); err != nil {
		return err
	}

	alg := make([]byte, length)
	if err := binary.Read(b, binary.BigEndian, &alg); err != nil {
		return err
	}

	p.Algorithm = string(alg)

	if err := binary.Read(b, binary.BigEndian, &length); err != nil {
		return err
	}

	p.KeyID = make([]byte, length)
	if err := binary.Read(b, binary.BigEndian, &p.KeyID); err != nil {
		return err
	}

	if err := binary.Read(b, binary.BigEndian, &length); err != nil {
		return err
	}

	rootKeyID := make([]byte, length)
	if err := binary.Read(b, binary.BigEndian, &rootKeyID); err != nil {
		return err
	}

	p.RootKeyID = string(rootKeyID)

	if err := binary.Read(b, binary.BigEndian, &length); err != nil {
		return err
	}

	p.Nonce = make([]byte, length)
	if err := binary.Read(b, binary.BigEndian, &p.Nonce); err != nil {
		return err
	}

	return nil
}

func marshalPayload(p *encryptedPayload) ([]byte, error) {
	b := bytes.NewBuffer(nil)

	if err := binary.Write(b, binary.BigEndian, uint32(len(p.Ciphertext))); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, p.Ciphertext); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, uint8(len(p.Algorithm))); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, []byte(p.Algorithm)); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, uint8(len(p.KeyID))); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, p.KeyID); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, uint8(len(p.RootKeyID))); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, []byte(p.RootKeyID)); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, uint8(len(p.Nonce))); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, p.Nonce); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
