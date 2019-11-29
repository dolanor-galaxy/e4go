// Package crypto defines the cryptographic functions used in E4
package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/agl/ed25519/extra25519"
	miscreant "github.com/miscreant/miscreant.go"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"
)

var (
	// ErrInvalidProtectedLen occurs when the protected message is  not of the expected length
	ErrInvalidProtectedLen = errors.New("invalid length of protected message")
	// ErrTooShortCipher occurs when trying to unprotect a cipher shorter than TimestampLen
	ErrTooShortCipher = errors.New("ciphertext too short")
	// ErrTimestampInFuture occurs when the cipher timestamp is in the future
	ErrTimestampInFuture = errors.New("timestamp received is in the future")
	// ErrTimestampTooOld occurs when the cipher timestamp is older than MaxDelayDuration from now
	ErrTimestampTooOld = errors.New("timestamp too old")
)

// Encrypt creates an authenticated ciphertext
func Encrypt(key, ad, pt []byte) ([]byte, error) {
	if err := ValidateSymKey(key); err != nil {
		return nil, err
	}

	// Use same key for CMAC and CTR, negligible security bound difference
	doublekey := append(key, key...)

	c, err := miscreant.NewAESCMACSIV(doublekey)
	if err != nil {
		return nil, err
	}
	ads := make([][]byte, 1)
	ads[0] = ad
	return c.Seal(nil, pt, ads...)
}

// Decrypt decrypts and verifies an authenticated ciphertext
func Decrypt(key, ad, ct []byte) ([]byte, error) {
	if err := ValidateSymKey(key); err != nil {
		return nil, err
	}

	// Use same key for CMAC and CTR, negligible security bound difference
	doublekey := append(key, key...)

	c, err := miscreant.NewAESCMACSIV(doublekey)
	if err != nil {
		return nil, err
	}
	if len(ct) < c.Overhead() {
		return nil, errors.New("too short ciphertext")
	}

	return c.Open(nil, ct, ad)
}

// ProtectCommandPubKey is an helper method to protect the given command using a client
// public key and a secret key
func ProtectCommandPubKey(command []byte, clientPubKey, secretKey *[32]byte) ([]byte, error) {
	var shared [32]byte
	curve25519.ScalarMult(&shared, secretKey, clientPubKey)

	key := Sha3Sum256(shared[:])[:KeyLen]

	return ProtectSymKey(command, key)
}

// DeriveSymKey derives a symmetric key from a password using Argon2
// (Replaces HashPwd)
func DeriveSymKey(pwd string) ([]byte, error) {
	if err := ValidatePassword(pwd); err != nil {
		return nil, fmt.Errorf("invalid password: %v", err)
	}

	return argon2.Key([]byte(pwd), nil, 1, 64*1024, 4, KeyLen), nil
}

// ProtectSymKey attempt to encrypt payload using given symmetric key
func ProtectSymKey(payload, key []byte) ([]byte, error) {
	timestamp := make([]byte, TimestampLen)
	binary.LittleEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	ct, err := Encrypt(key, timestamp, payload)
	if err != nil {
		return nil, err
	}
	protected := append(timestamp, ct...)

	protectedLen := TimestampLen + len(payload) + TagLen
	if protectedLen != len(protected) {
		return nil, ErrInvalidProtectedLen
	}

	return protected, nil
}

// UnprotectSymKey attempt to decrypt protected bytes, using given symmetric key
func UnprotectSymKey(protected, key []byte) ([]byte, error) {
	if len(protected) <= TimestampLen+TagLen {
		return nil, ErrTooShortCipher
	}

	ct := protected[TimestampLen:]
	timestamp := protected[:TimestampLen]

	if err := ValidateTimestamp(timestamp); err != nil {
		return nil, err
	}

	pt, err := Decrypt(key, timestamp, ct)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

// RandomKey generates a random KeyLen-byte key usable by Encrypt and Decrypt
func RandomKey() []byte {
	key := make([]byte, KeyLen)
	n, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	if n != KeyLen {
		panic(fmt.Errorf("bytes read mismatch in RandomKey: got %d wanted %d", n, KeyLen))
	}

	return key
}

// RandomID generates a random IDLen-byte ID
func RandomID() []byte {
	id := make([]byte, IDLen)
	n, err := rand.Read(id)
	if err != nil {
		panic(err)
	}
	if n != IDLen {
		panic(fmt.Errorf("bytes read mismatch in RandomID: got %d wanted %d", n, IDLen))
	}

	return id
}

// RandomDelta16 produces a random 16-bit integer to allow us to
// vary key sizes, plaintext sizes etc
func RandomDelta16() uint16 {
	randAdjust := make([]byte, 2)
	rand.Read(randAdjust)
	return binary.LittleEndian.Uint16(randAdjust)
}

// Ed25519PrivateKeyFromPassword creates a ed25519.PrivateKey from a password
func Ed25519PrivateKeyFromPassword(password string) (ed25519.PrivateKey, error) {
	if err := ValidatePassword(password); err != nil {
		return nil, fmt.Errorf("invalid password: %v", err)
	}

	seed := argon2.Key([]byte(password), nil, 1, 64*1024, 4, ed25519.SeedSize)
	return ed25519.NewKeyFromSeed(seed), nil
}

// PublicEd25519KeyToCurve25519 convert an ed25519.PublicKey to a curve25519 public key.
func PublicEd25519KeyToCurve25519(edPubKey ed25519.PublicKey) [32]byte {
	var edPk [32]byte
	var curveKey [32]byte
	copy(edPk[:], edPubKey)
	if !extra25519.PublicKeyToCurve25519(&curveKey, &edPk) {
		panic("could not convert ed25519 public key to curve25519")
	}

	return curveKey
}

// PrivateEd25519KeyToCurve25519 convert an ed25519.PrivateKey to a curve25519 private key.
func PrivateEd25519KeyToCurve25519(edPrivKey ed25519.PrivateKey) [32]byte {
	var edSk [64]byte
	var curveKey [32]byte
	copy(edSk[:], edPrivKey)
	extra25519.PrivateKeyToCurve25519(&curveKey, &edSk)

	return curveKey
}
