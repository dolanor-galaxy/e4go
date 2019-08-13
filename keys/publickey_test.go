package keys

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/agl/ed25519/extra25519"

	e4crypto "gitlab.com/teserakt/e4common/crypto"
	"golang.org/x/crypto/ed25519"
)

func TestNewPubKeyMaterialKey(t *testing.T) {
	expectedSignerID := e4crypto.HashIDAlias("test")
	_, expectedPrivateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate ed25519 private key: %v", err)
	}

	expectedC2PubKey := getTestC2PublicKey(t)

	key, err := NewPubKeyMaterialKey(expectedSignerID, expectedPrivateKey, expectedC2PubKey)
	if err != nil {
		t.Fatalf("expected no error creating a key, got %v", err)
	}

	assertPubKeyMaterialKeyContains(t, key, expectedSignerID, expectedPrivateKey, expectedC2PubKey)

	invalidSignerID := make([]byte, e4crypto.IDLen-1)
	_, err = NewPubKeyMaterialKey(invalidSignerID, expectedPrivateKey, expectedC2PubKey)
	if err == nil {
		t.Fatal("expected invalid signerID to produce an error")
	}

	invalidPrivateKey := make([]byte, len(expectedPrivateKey))
	_, err = NewPubKeyMaterialKey(expectedSignerID, invalidPrivateKey, expectedC2PubKey)
	if err == nil {
		t.Fatal("expected invalid private key to produce an error")
	}
}

func TestNewPubKeyMaterialKeyFromPassword(t *testing.T) {
	password := "test-password-random"

	expectedSignerID := e4crypto.HashIDAlias("test")
	expectedC2PubKey := getTestC2PublicKey(t)

	key, err := NewPubKeyMaterialKeyFromPassword(expectedSignerID, password, expectedC2PubKey)
	if err != nil {
		t.Fatalf("expected no error creating a key, got %v", err)
	}

	expectedPrivateKey, err := e4crypto.Ed25519PrivateKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create key from password: %v", err)
	}

	assertPubKeyMaterialKeyContains(t, key, expectedSignerID, expectedPrivateKey, expectedC2PubKey)
}

func assertPubKeyMaterialKeyContains(
	t *testing.T,
	key PubKeyMaterialKey,
	expectedSignerID []byte,
	expectedPrivateKey ed25519.PrivateKey,
	expectedC2PubKey [32]byte,
) {
	tkey, ok := key.(*pubKeyMaterialKey)
	if !ok {
		t.Fatal("failed to cast key")
	}

	if bytes.Equal(tkey.SignerID, expectedSignerID) == false {
		t.Fatalf("expected signerID to be %v, got %v", expectedSignerID, tkey.SignerID)
	}

	if tkey.C2PublicKey != expectedC2PubKey {
		t.Fatalf("expected c2PublicKey to be %v, got %v", expectedC2PubKey, tkey.C2PublicKey)
	}

	if bytes.Equal(tkey.PrivateKey, expectedPrivateKey) == false {
		t.Fatalf("expected private key to be %v, got %v", expectedPrivateKey, tkey.PrivateKey)
	}
}

func getTestC2PublicKey(t *testing.T) [32]byte {
	pubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate ed25519 public key: %v", err)
	}
	var expectedC2PubKey [32]byte
	copy(expectedC2PubKey[:], pubKey[:32])

	return expectedC2PubKey
}

func TestNewRandomPubKeyMaterialKey(t *testing.T) {
	expectedSignerID := e4crypto.HashIDAlias("test")
	expectedC2PubKey := getTestC2PublicKey(t)

	key, err := NewRandomPubKeyMaterialKey(expectedSignerID, expectedC2PubKey)
	if err != nil {
		t.Fatalf("expected no error creating a key, got %v", err)
	}

	tkey, ok := key.(*pubKeyMaterialKey)
	if !ok {
		t.Fatal("failed to cast key")
	}

	if bytes.Equal(tkey.SignerID, expectedSignerID) == false {
		t.Fatalf("expected signerID to be %v, got %v", expectedSignerID, tkey.SignerID)
	}

	if tkey.C2PublicKey != expectedC2PubKey {
		t.Fatalf("expected c2PublicKey to be %v, got %v", expectedC2PubKey, tkey.C2PublicKey)
	}

	if err := e4crypto.ValidateEd25519PrivKey(tkey.PrivateKey); err != nil {
		t.Fatalf("expected a valid private key to be set: got error: %v", err)
	}
}

func TestPubKeyMaterialKeyProtectUnprotectMessage(t *testing.T) {
	clientID := e4crypto.HashIDAlias("test")
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keys: %v", err)
	}

	k, err := NewPubKeyMaterialKey(clientID, privKey, getTestC2PublicKey(t))
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	payload := []byte("some message")
	topicKey := e4crypto.RandomKey()

	protected, err := k.ProtectMessage(payload, topicKey)
	if err != nil {
		t.Fatalf("failed to protect message: %v", err)
	}

	_, err = k.UnprotectMessage(protected, topicKey)
	if err == nil {
		t.Fatal("expected unprotect to fail without the proper public key")
	}

	k.AddPubKey(clientID, pubKey)
	unprotected, err := k.UnprotectMessage(protected, topicKey)
	if err != nil {
		t.Fatalf("expected no error when unprotecting message, got %v", err)
	}

	if bytes.Equal(unprotected, payload) == false {
		t.Fatalf("expected unprotected message to be %v, got %v", unprotected, payload)
	}

	badTopicKey := e4crypto.RandomKey()
	_, err = k.UnprotectMessage(protected, badTopicKey)
	if err == nil {
		t.Fatal("expected unprotect to fail without the proper topic key")
	}

	if _, err := k.UnprotectMessage([]byte("too short"), topicKey); err == nil {
		t.Fatal("expected unprotect to fail with a too short protected message")
	}

	if _, err := k.ProtectMessage([]byte("some message"), []byte("not a key")); err == nil {
		t.Fatal("expected protect message to fail with a bad topic key")
	}

	tooOldProtected := make([]byte, len(protected))
	copy(tooOldProtected, protected)

	tooOldTs := make([]byte, e4crypto.TimestampLen)
	binary.LittleEndian.PutUint64(tooOldTs, uint64(time.Now().Add(-(e4crypto.MaxSecondsDelay+1)*time.Second).Unix()))

	tooOldProtected = append(tooOldTs, tooOldProtected[e4crypto.TimestampLen:]...)
	if _, err := k.UnprotectMessage(tooOldProtected, topicKey); err == nil {
		t.Fatal("expected unprotect message to fail with a too old timestamp")
	}
}

func TestPubKeyMaterialKeyUnprotectCommand(t *testing.T) {
	clientID := e4crypto.HashIDAlias("test")
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keys: %v", err)
	}

	c2PubKey, c2PrivateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate c2 secret key: %v", err)
	}
	var c2EdPk [32]byte
	copy(c2EdPk[:], c2PubKey[:32])
	var c2Pk [32]byte

	var c2EdSk [64]byte
	copy(c2EdSk[:], c2PrivateKey)

	var c2Sk [32]byte
	extra25519.PublicKeyToCurve25519(&c2Pk, &c2EdPk)
	extra25519.PrivateKeyToCurve25519(&c2Sk, &c2EdSk)

	k, err := NewPubKeyMaterialKey(clientID, privKey, c2Pk)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	command := []byte{0x01, 0x02, 0x03, 0x04}

	var clientEdPk [32]byte
	var clientPk [32]byte
	copy(clientEdPk[:], pubKey[:32])
	extra25519.PublicKeyToCurve25519(&clientPk, &clientEdPk)

	protectedCmd, err := e4crypto.ProtectCommandPubKey(command, &clientPk, &c2Sk)
	if err != nil {
		t.Fatalf("failed to protect command: %v", err)
	}

	unprotectedCmd, err := k.UnprotectCommand(protectedCmd)
	if err != nil {
		t.Fatalf("expected no error when unprotecting command, got %v", err)
	}

	if bytes.Equal(unprotectedCmd, command) == false {
		t.Fatalf("expected unprotected command to be %v, got %v", command, unprotectedCmd)
	}
}

func TestPubKeyMaterialKeyPubKeys(t *testing.T) {
	clientID := e4crypto.HashIDAlias("test")

	k, err := NewRandomPubKeyMaterialKey(clientID, getTestC2PublicKey(t))
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	if c := len(k.GetPubKeys()); c != 0 {
		t.Fatalf("expected pubkeys length to be 0, got %d", c)
	}

	pk0, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate public key: %v", err)
	}
	if err := k.AddPubKey([]byte("id1"), pk0); err != nil {
		t.Fatalf("expected no error when adding pubkey for id1, got: %v", err)
	}

	pk, err := k.GetPubKey([]byte("id1"))
	if err != nil {
		t.Fatalf("failed to get pubKey: %v", err)
	}
	if bytes.Equal(pk0, pk) == false {
		t.Fatalf("expected id1 pubkey to be %v, got %v", pk0, pk)
	}

	pk1, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate public key: %v", err)
	}

	if err := k.AddPubKey([]byte("id1"), pk1); err != nil {
		t.Fatalf("expected no error when adding pubkey for id1, got: %v", err)
	}

	if c := len(k.GetPubKeys()); c != 1 {
		t.Fatalf("expected pubkeys length to be 1, got %d", c)
	}

	pk, err = k.GetPubKey([]byte("id1"))
	if err != nil {
		t.Fatalf("failed to get pubKey: %v", err)
	}
	if bytes.Equal(pk1, pk) == false {
		t.Fatalf("expected id1 pubkey to be %v, got %v", pk1, pk)
	}

	pk2, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate public key: %v", err)
	}

	if err := k.AddPubKey([]byte("id2"), pk2); err != nil {
		t.Fatalf("expected no error when adding pubkey for id2, got: %v", err)
	}

	if c := len(k.GetPubKeys()); c != 2 {
		t.Fatalf("expected pubkeys length to be 2, got %d", c)
	}

	pk, err = k.GetPubKey([]byte("id1"))
	if err != nil {
		t.Fatalf("failed to get public key: %v", err)
	}
	if bytes.Equal(pk1, pk) == false {
		t.Fatalf("expected id1 pubkey to be %v, got %v", pk1, pk)
	}

	pk, err = k.GetPubKey([]byte("id2"))
	if err != nil {
		t.Fatalf("failed to get public key: %v", err)
	}
	if bytes.Equal(pk2, pk) == false {
		t.Fatalf("expected id2 pubkey to be %v, got %v", pk2, pk)
	}

	if err := k.RemovePubKey([]byte("id1")); err != nil {
		t.Fatalf("failed to remove pubkey for id1: %v", err)
	}
	if c := len(k.GetPubKeys()); c != 1 {
		t.Fatalf("expected pubkeys length to be 1, got %d", c)
	}

	pk, err = k.GetPubKey([]byte("id2"))
	if err != nil {
		t.Fatalf("failed to get public key: %v", err)
	}
	if bytes.Equal(pk2, pk) == false {
		t.Fatalf("expected id2 pubkey to be %v, got %v", pk2, pk)
	}

	if _, err := k.GetPubKey([]byte("id1")); err != ErrPubKeyNotFound {
		t.Fatal("expected pubkey for id1 to be removed")
	}

	// Double remove must return an error
	if err := k.RemovePubKey([]byte("id1")); err == nil {
		t.Fatal("expected an error when removing an inexisting pubKey")
	}

	// Reset clears all
	k.ResetPubKeys()
	if c := len(k.GetPubKeys()); c != 0 {
		t.Fatalf("expected reset to have removed all keys, got %d remaining", c)
	}
	if _, err := k.GetPubKey([]byte("id2")); err != ErrPubKeyNotFound {
		t.Fatal("expected pubkey for id2 to be removed")
	}

	// Adding invalid keys return errors
	if err := k.AddPubKey([]byte("id1"), []byte("not a key")); err == nil {
		t.Fatal("expected an error when adding an invalid pubKey")
	}
}

func TestPubKeyMaterialKeySetKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	clientID := e4crypto.HashIDAlias("test")

	k, err := NewPubKeyMaterialKey(clientID, privateKey, getTestC2PublicKey(t))
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	tkey, ok := k.(*pubKeyMaterialKey)
	if !ok {
		t.Fatal("failed to cast key")
	}

	if bytes.Equal(tkey.PrivateKey, privateKey) == false {
		t.Fatalf("expected private key to be %v, got %v", privateKey, tkey.PrivateKey)
	}

	_, privateKey2, err := ed25519.GenerateKey(nil)
	if err := tkey.SetKey(privateKey2); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	if bytes.Equal(tkey.PrivateKey, privateKey2) == false {
		t.Fatalf("expected private key to be %v, got %v", privateKey2, tkey.PrivateKey)
	}

	if err := tkey.SetKey([]byte("not a key")); err == nil {
		t.Fatal("expected SetKey with invalid key to returns an error")
	}

	privateKey2[0] = privateKey2[0] + 1
	if bytes.Equal(tkey.PrivateKey, privateKey2) == true {
		t.Fatalf("expected private key to have been copied, seems still pointing to same slice")
	}
}

func TestPubKeyMaterialKeyMarshalJSON(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	clientID := e4crypto.HashIDAlias("test")
	c2Pk := getTestC2PublicKey(t)

	k, err := NewPubKeyMaterialKey(clientID, privateKey, c2Pk)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	pk1, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate public key: %v", err)
	}
	if err := k.AddPubKey([]byte("id1"), pk1); err != nil {
		t.Fatalf("expected no error when adding pubkey for id1, got: %v", err)
	}

	pk2, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate public key: %v", err)
	}
	if err := k.AddPubKey([]byte("id2"), pk2); err != nil {
		t.Fatalf("expected no error when adding pubkey for id2, got: %v", err)
	}

	jsonKey, err := k.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal key into json: %v", err)
	}

	unmarshalledKey, err := FromRawJSON(jsonKey)
	if err != nil {
		t.Fatalf("failed to unmarshal json key: %v", err)
	}

	if reflect.DeepEqual(unmarshalledKey, k) == false {
		t.Fatalf("expected unmarshalled key to be %#v, got %#v", k, unmarshalledKey)
	}
}
