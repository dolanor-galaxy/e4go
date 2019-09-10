package keys

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/agl/ed25519/extra25519"

	e4crypto "github.com/teserakt-io/e4common/crypto"
	"golang.org/x/crypto/ed25519"
)

func TestNewPubKeyMaterial(t *testing.T) {
	expectedSignerID := e4crypto.HashIDAlias("test")
	_, expectedPrivateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 private key: %v", err)
	}

	expectedC2PubKey := getTestC2PubKey(t)

	key, err := NewPubKeyMaterial(expectedSignerID, expectedPrivateKey, expectedC2PubKey)
	if err != nil {
		t.Fatalf("Key creation failed: %v", err)
	}

	assertPubKeyMaterialContains(t, key, expectedSignerID, expectedPrivateKey, expectedC2PubKey)

	invalidSignerID := make([]byte, e4crypto.IDLen-1)
	_, err = NewPubKeyMaterial(invalidSignerID, expectedPrivateKey, expectedC2PubKey)
	if err == nil {
		t.Fatal("Expected an invalid signerID to produce an error when creating a key material")
	}

	invalidPrivateKey := make([]byte, len(expectedPrivateKey))
	_, err = NewPubKeyMaterial(expectedSignerID, invalidPrivateKey, expectedC2PubKey)
	if err == nil {
		t.Fatal("Expected an invalid private key to produce an error when creating a key material")
	}
}

func TestNewPubKeyMaterialFromPassword(t *testing.T) {
	password := "test-password-random"

	expectedSignerID := e4crypto.HashIDAlias("test")
	expectedC2PubKey := getTestC2PubKey(t)

	key, err := NewPubKeyMaterialFromPassword(expectedSignerID, password, expectedC2PubKey)
	if err != nil {
		t.Fatalf("Key creation failed: %v", err)
	}

	expectedPrivateKey, err := e4crypto.Ed25519PrivateKeyFromPassword(password)
	if err != nil {
		t.Fatalf("Key creation from password failed: %v", err)
	}

	assertPubKeyMaterialContains(t, key, expectedSignerID, expectedPrivateKey, expectedC2PubKey)
}

func assertPubKeyMaterialContains(
	t *testing.T,
	key PubKeyMaterial,
	expectedSignerID []byte,
	expectedPrivateKey ed25519.PrivateKey,
	expectedC2PubKey []byte,
) {
	typedKey, ok := key.(*pubKeyMaterial)
	if !ok {
		t.Fatalf("Unexpected type: got %T, wanted pubKeyMaterial", key)
	}

	if !bytes.Equal(typedKey.SignerID, expectedSignerID) {
		t.Fatalf("Invalid signer ID: got %v, wanted %v", typedKey.SignerID, expectedSignerID)
	}

	if !bytes.Equal(typedKey.C2PubKey, expectedC2PubKey) {
		t.Fatalf("Invalid c2PubKey: got %v, wanted %v", typedKey.C2PubKey, expectedC2PubKey)
	}

	if !bytes.Equal(typedKey.PrivateKey, expectedPrivateKey) {
		t.Fatalf("Invalid private key: got %v, wanted %v", typedKey.PrivateKey, expectedPrivateKey)
	}
}

func getTestC2PubKey(t *testing.T) []byte {
	pubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 public key: %v", err)
	}

	return pubKey
}

func TestNewRandomPubKeyMaterial(t *testing.T) {
	expectedSignerID := e4crypto.HashIDAlias("test")
	expectedC2PubKey := getTestC2PubKey(t)

	key, err := NewRandomPubKeyMaterial(expectedSignerID, expectedC2PubKey)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	typedKey, ok := key.(*pubKeyMaterial)
	if !ok {
		t.Fatalf("Unexpected type: got %T, wanted pubKeyMaterial", key)
	}

	if !bytes.Equal(typedKey.SignerID, expectedSignerID) {
		t.Fatalf("Invalid signerID: got %v, wanted: %v", typedKey.SignerID, expectedSignerID)
	}

	if !bytes.Equal(typedKey.C2PubKey, expectedC2PubKey) {
		t.Fatalf("Invalid c2PubKey: got %v, wanted %v", typedKey.C2PubKey, expectedC2PubKey)
	}

	if err := e4crypto.ValidateEd25519PrivKey(typedKey.PrivateKey); err != nil {
		t.Fatalf("Failed to validate private key: %v", err)
	}
}

func TestPubKeyMaterialProtectUnprotectMessage(t *testing.T) {
	clientID := e4crypto.HashIDAlias("test")
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 keys: %v", err)
	}

	k, err := NewPubKeyMaterial(clientID, privKey, getTestC2PubKey(t))
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	payload := []byte("some message")
	topicKey := e4crypto.RandomKey()

	protected, err := k.ProtectMessage(payload, topicKey)
	if err != nil {
		t.Fatalf("Failed to protect message: %v", err)
	}

	_, err = k.UnprotectMessage(protected, topicKey)
	if err == nil {
		t.Fatal("Expected unprotect to fail without the proper public key")
	}

	k.AddPubKey(clientID, pubKey)
	unprotected, err := k.UnprotectMessage(protected, topicKey)
	if err != nil {
		t.Fatalf("Failed to unprotect message: %v", err)
	}

	if !bytes.Equal(unprotected, payload) {
		t.Fatalf("Invalid unprotected message: got %v, wanted: %v", unprotected, payload)
	}

	badTopicKey := e4crypto.RandomKey()
	_, err = k.UnprotectMessage(protected, badTopicKey)
	if err == nil {
		t.Fatal("Expected unprotect to fail without the proper topic key")
	}

	if _, err := k.UnprotectMessage([]byte("too short"), topicKey); err == nil {
		t.Fatal("Expected unprotect to fail with a too short protected message")
	}

	if _, err := k.ProtectMessage([]byte("some message"), []byte("not a key")); err == nil {
		t.Fatal("Expected protect message to fail with a bad topic key")
	}

	tooOldProtected := make([]byte, len(protected))
	copy(tooOldProtected, protected)

	tooOldTs := make([]byte, e4crypto.TimestampLen)
	binary.LittleEndian.PutUint64(tooOldTs, uint64(time.Now().Add(-(e4crypto.MaxDelayDuration + 1)).Unix()))

	tooOldProtected = append(tooOldTs, tooOldProtected[e4crypto.TimestampLen:]...)
	if _, err := k.UnprotectMessage(tooOldProtected, topicKey); err == nil {
		t.Fatal("Expected unprotect message to fail with a too old timestamp")
	}
}

func TestPubKeyMaterialUnprotectCommand(t *testing.T) {
	clientID := e4crypto.HashIDAlias("test")
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 keys: %v", err)
	}

	c2PubKey, c2PrivateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate c2 secret key: %v", err)
	}
	var c2EdPk [32]byte
	copy(c2EdPk[:], c2PubKey[:32])
	var c2Pk [32]byte

	var c2EdSk [64]byte
	copy(c2EdSk[:], c2PrivateKey)

	var c2Sk [32]byte
	extra25519.PublicKeyToCurve25519(&c2Pk, &c2EdPk)
	extra25519.PrivateKeyToCurve25519(&c2Sk, &c2EdSk)

	k, err := NewPubKeyMaterial(clientID, privKey, c2Pk[:])
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	command := []byte{0x01, 0x02, 0x03, 0x04}

	var clientEdPk [32]byte
	var clientPk [32]byte
	copy(clientEdPk[:], pubKey[:32])
	extra25519.PublicKeyToCurve25519(&clientPk, &clientEdPk)

	protectedCmd, err := e4crypto.ProtectCommandPubKey(command, &clientPk, &c2Sk)
	if err != nil {
		t.Fatalf("Failed to protect command: %v", err)
	}

	unprotectedCmd, err := k.UnprotectCommand(protectedCmd)
	if err != nil {
		t.Fatalf("Failed to unprotect command: %v", err)
	}

	if !bytes.Equal(unprotectedCmd, command) {
		t.Fatalf("Invalid unprotected command: got %v, wanted %v", unprotectedCmd, command)
	}
}

func TestPubKeyMaterialPubKeys(t *testing.T) {
	clientID := e4crypto.HashIDAlias("test")

	k, err := NewRandomPubKeyMaterial(clientID, getTestC2PubKey(t))
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	if c := len(k.GetPubKeys()); c != 0 {
		t.Fatalf("Invalid pubkey count: got %d, wanted 0", c)
	}

	pk0, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate public key: %v", err)
	}
	if err := k.AddPubKey([]byte("id1"), pk0); err != nil {
		t.Fatalf("Failed to add pubkey for id1: %v", err)
	}

	pk, err := k.GetPubKey([]byte("id1"))
	if err != nil {
		t.Fatalf("Failed to get pubKey: %v", err)
	}
	if !bytes.Equal(pk, pk0) {
		t.Fatalf("Invalid pubKey for id1: got %v, wanted %v", pk, pk0)
	}

	pk1, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate public key: %v", err)
	}

	if err := k.AddPubKey([]byte("id1"), pk1); err != nil {
		t.Fatalf("Failed to add pubkey for id1: %v", err)
	}

	if c := len(k.GetPubKeys()); c != 1 {
		t.Fatalf("Invalid pubkey count: got %d, wanted 1", c)
	}

	pk, err = k.GetPubKey([]byte("id1"))
	if err != nil {
		t.Fatalf("Failed to get pubKey: %v", err)
	}
	if !bytes.Equal(pk, pk1) {
		t.Fatalf("Invalid pubkey for id1: got %v, wanted %v", pk, pk1)
	}

	pk2, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate public key: %v", err)
	}

	if err := k.AddPubKey([]byte("id2"), pk2); err != nil {
		t.Fatalf("Failed to add pubkey for id2: %v", err)
	}

	if c := len(k.GetPubKeys()); c != 2 {
		t.Fatalf("Invalid pubkey count: got %d, wanted 2", c)
	}

	pk, err = k.GetPubKey([]byte("id1"))
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}
	if !bytes.Equal(pk, pk1) {
		t.Fatalf("Invalid pubkey for id1: got %v, wanted %v", pk, pk1)
	}

	pk, err = k.GetPubKey([]byte("id2"))
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}
	if !bytes.Equal(pk, pk2) {
		t.Fatalf("Invalid pubkey for id2: got %v, wanted %v", pk, pk2)
	}

	if err := k.RemovePubKey([]byte("id1")); err != nil {
		t.Fatalf("Failed to remove pubkey for id1: %v", err)
	}
	if c := len(k.GetPubKeys()); c != 1 {
		t.Fatalf("Invalid pubkey count: got %d, wanted 1", c)
	}

	pk, err = k.GetPubKey([]byte("id2"))
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}
	if !bytes.Equal(pk, pk2) {
		t.Fatalf("Invalid pubkey for id2: got %v, wanted %v", pk, pk2)
	}

	if _, err := k.GetPubKey([]byte("id1")); err != ErrPubKeyNotFound {
		t.Fatal("Expected pubkey for id1 to be removed")
	}

	// Double remove must return an error
	if err := k.RemovePubKey([]byte("id1")); err == nil {
		t.Fatal("Expected an error when removing an inexisting pubKey")
	}

	// Reset clears all
	k.ResetPubKeys()
	if c := len(k.GetPubKeys()); c != 0 {
		t.Fatalf("Invalid pubkey count: got %d, wanted 0", c)
	}
	if _, err := k.GetPubKey([]byte("id2")); err != ErrPubKeyNotFound {
		t.Fatal("Expected pubkey for id2 to be removed")
	}

	// Adding invalid keys return errors
	if err := k.AddPubKey([]byte("id1"), []byte("not a key")); err == nil {
		t.Fatal("Expected an error when adding an invalid pubKey")
	}
}

func TestPubKeyMaterialSetKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	clientID := e4crypto.HashIDAlias("test")

	k, err := NewPubKeyMaterial(clientID, privateKey, getTestC2PubKey(t))
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	typedKey, ok := k.(*pubKeyMaterial)
	if !ok {
		t.Fatalf("Unexpected type: got %T, wanted pubKeyMaterial", k)
	}

	if !bytes.Equal(typedKey.PrivateKey, privateKey) {
		t.Fatalf("Invalid private key: got %v, wanted %v", typedKey.PrivateKey, privateKey)
	}

	_, privateKey2, err := ed25519.GenerateKey(nil)
	if err := typedKey.SetKey(privateKey2); err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	if !bytes.Equal(typedKey.PrivateKey, privateKey2) {
		t.Fatalf("Invalid private key: got %v, wanted %v", typedKey.PrivateKey, privateKey2)
	}

	if err := typedKey.SetKey([]byte("not a key")); err == nil {
		t.Fatal("Expected SetKey with invalid key to returns an error")
	}

	privateKey2[0] = privateKey2[0] + 1
	if bytes.Equal(typedKey.PrivateKey, privateKey2) {
		t.Fatalf("Expected private key slice to have been copied, but it is still pointing to same slice")
	}
}

func TestPubKeyMaterialMarshalJSON(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	clientID := e4crypto.HashIDAlias("test")
	c2Pk := getTestC2PubKey(t)

	k, err := NewPubKeyMaterial(clientID, privateKey, c2Pk)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	pk1, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate public key: %v", err)
	}
	if err := k.AddPubKey([]byte("id1"), pk1); err != nil {
		t.Fatalf("Failed to add pubkey for id1: %v", err)
	}

	pk2, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate public key: %v", err)
	}
	if err := k.AddPubKey([]byte("id2"), pk2); err != nil {
		t.Fatalf("Failed to add pubkey for id2: %v", err)
	}

	jsonKey, err := json.Marshal(k)
	if err != nil {
		t.Fatalf("Failed to marshal key into json: %v", err)
	}

	unmarshalledKey, err := FromRawJSON(jsonKey)
	if err != nil {
		t.Fatalf("Failed to unmarshal json key: %v", err)
	}

	if !reflect.DeepEqual(unmarshalledKey, k) {
		t.Fatalf("Invalid unmarshalled key: got %v, wanted %v", unmarshalledKey, k)
	}
}
