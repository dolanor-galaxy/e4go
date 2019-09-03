package e4common

import (
	"log"

	"github.com/agl/ed25519/extra25519"
	"golang.org/x/crypto/ed25519"

	"gitlab.com/teserakt/e4common/crypto"
)

func ExampleNewSymKeyClient() {
	client, err := NewSymKeyClient([]byte("clientID"), crypto.RandomKey(), "./symClient.json")
	if err != nil {
		panic(err)
	}

	protectedMessage, err := client.ProtectMessage([]byte("very secret message"), "topic/name")
	if err != nil {
		panic(err)
	}

	log.Printf("Protected message: %v", protectedMessage)
}

func ExampleNewSymKeyClientPretty() {
	client, err := NewSymKeyClientPretty("clientName", "verySecretPassword", "./symClient.json")
	if err != nil {
		panic(err)
	}

	protectedMessage, err := client.ProtectMessage([]byte("very secret message"), "topic/name")
	if err != nil {
		panic(err)
	}

	log.Printf("Protected message: %v", protectedMessage)
}

func ExampleNewPubKeyClient() {
	privateKey, err := crypto.Ed25519PrivateKeyFromPassword("verySecretPassword")
	if err != nil {
		panic(err)
	}

	var c2PubKey [32]byte
	c2EdPubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	var c2EdPk [32]byte
	copy(c2EdPk[:], c2EdPubKey)
	extra25519.PublicKeyToCurve25519(&c2PubKey, &c2EdPk)

	client, err := NewPubKeyClient([]byte("clientID"), privateKey, "./pubClient.json", c2PubKey[:])
	if err != nil {
		panic(err)
	}

	protectedMessage, err := client.ProtectMessage([]byte("very secret message"), "topic/name")
	if err != nil {
		panic(err)
	}

	log.Printf("Protected message: %v", protectedMessage)
}

func ExampleNewPubKeyClientPretty() {
	var c2PubKey [32]byte
	c2EdPubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	var c2EdPk [32]byte
	copy(c2EdPk[:], c2EdPubKey)
	extra25519.PublicKeyToCurve25519(&c2PubKey, &c2EdPk)

	client, err := NewPubKeyClientPretty("clientName", "verySecretPassword", "./pubClient.json", c2PubKey[:])
	if err != nil {
		panic(err)
	}

	protectedMessage, err := client.ProtectMessage([]byte("very secret message"), "topic/name")
	if err != nil {
		panic(err)
	}

	log.Printf("Protected message: %v", protectedMessage)
}
