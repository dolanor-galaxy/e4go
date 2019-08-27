package crypto

import (
	"encoding/hex"
	"testing"
)

/* TestHash tests KATs for both the hash function of choice and
 * the password hashing function / KDF of choice */
func TestHash(t *testing.T) {
	h := hex.EncodeToString(HashIDAlias("abc"))
	expected := "3a985da74fe225b2045c172d6bd390bd"
	if h != expected {
		t.Fatalf("Hash of ID alias incorrect, got: %s, wanted: %s", h, expected)
	}

	k, err := DeriveSymKey("testRandomPassword")
	if err != nil {
		t.Fatalf("Failed to derive symkey: %v", err)
	}

	h = hex.EncodeToString(k)
	expected = "ae153aa9dad7a10b0aed6d5bcfb407c77066acfbb2eaa702a6a88b6cf1b88c33"
	if h != expected {
		t.Fatalf("Hash of password incorrect, got: %s, wanted: %s", h, expected)
	}

	h = hex.EncodeToString(HashTopic("abc"))
	expected = "3a985da74fe225b2045c172d6bd390bd"
	if h != expected {
		t.Fatalf("Hash of Topic incorrect, got: %s, wanted: %s", h, expected)
	}
}
