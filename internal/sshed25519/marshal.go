package sshed25519

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"

	"golang.org/x/crypto/ssh"
)

// MarshalED25519PrivateKey marshals an ed25519 private key in the OpenSSH
// format.
//
// Copied with modification from https://github.com/mikesmitty/edkey.
//
// See https://github.com/golang/go/issues/37132 for an approved, but stalled,
// proposal to add a similar function to the stdlib.
func MarshalED25519PrivateKey(key ed25519.PrivateKey) ([]byte, error) {
	checkRaw, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32+1))
	if err != nil {
		return nil, fmt.Errorf("generate random check: %w", err)
	}
	// Convert it to uint32. As the content was within uint32 limits, nothing should be lost
	check := uint32(checkRaw.Uint64())

	pk1 := struct {
		Check1  uint32
		Check2  uint32
		Keytype string
		Pub     []byte
		Priv    []byte
		Comment string
		Pad     []byte `ssh:"rest"`
	}{
		Check1:  check,
		Check2:  check,
		Keytype: ssh.KeyAlgoED25519,
		Priv:    key,
	}

	pubKey := []byte(key.Public().(ed25519.PublicKey)) // nolint:forcetypeassert
	pk1.Pub = pubKey

	// Add some padding to match the encryption block size within PrivKeyBlock (without Pad field)
	// 8 doesn't match the documentation, but that's what ssh-keygen uses for unencrypted keys. *shrug*
	bs := 8
	blockLen := len(ssh.Marshal(pk1))
	padLen := (bs - (blockLen % bs)) % bs
	pk1.Pad = make([]byte, padLen)

	// Padding is a sequence of bytes like: 1, 2, 3...
	for i := 0; i < padLen; i++ {
		pk1.Pad[i] = byte(i + 1)
	}

	// Generate the pubkey prefix "\0\0\0\nssh-ed25519\0\0\0 "
	pubKeyBytes := []byte{0x0, 0x0, 0x0, 0x0b}
	pubKeyBytes = append(pubKeyBytes, []byte(ssh.KeyAlgoED25519)...)
	pubKeyBytes = append(pubKeyBytes, []byte{0x0, 0x0, 0x0, 0x20}...)
	pubKeyBytes = append(pubKeyBytes, pubKey...)

	var header struct {
		CipherName   string
		KdfName      string
		KdfOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}
	header.CipherName = "none"
	header.KdfName = "none"
	header.KdfOpts = ""
	header.NumKeys = 1
	header.PubKey = pubKeyBytes
	header.PrivKeyBlock = ssh.Marshal(pk1)

	magic := append([]byte("openssh-key-v1"), 0)
	magic = append(magic, ssh.Marshal(header)...)

	return magic, nil
}
