package xpff

import (
	"testing"
)

func TestXPFFHeaderGenerator_RoundTrip(t *testing.T) {
	baseKey := "0e6be1f1e21ffc33590b888fd4dc81b19713e570e805d4e5df80a493c9571a05"
	gen := XPFFHeaderGenerator{baseKey: baseKey}

	guestID := "v1%3A174849298500261196"
	plaintext := `{"navigator_properties":{"hasBeenActive":"true","userAgent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36","webdriver":"false"},"created_at":1750014202073}`

	// Encrypt
	encrypted, err := gen.GenerateXPFF(plaintext, guestID)
	if err != nil {
		t.Fatalf("GenerateXPFF returned error: %v", err)
	}
	if len(encrypted) == 0 {
		t.Fatal("GenerateXPFF returned empty string")
	}
	t.Logf("Encrypted: %s", encrypted)

	// Decrypt
	decrypted, err := gen.DecodeXPFF(encrypted, guestID)
	if err != nil {
		t.Fatalf("DecodeXPFF returned error: %v", err)
	}
	t.Logf("Decrypted: %s", decrypted)

	// Verify round-trip
	if decrypted != plaintext {
		t.Errorf("Round-trip mismatch:\n got:  %q\n want: %q", decrypted, plaintext)
	}
}
