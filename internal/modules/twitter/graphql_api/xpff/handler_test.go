// xpff_handler_test.go
package xpff

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"
)

func TestHandler_GetXPFFHeader_RoundTrip(t *testing.T) {
	guestID := "v1:174864550766043618"
	userAgent := "test-agent/1.0"
	handler := NewHandler(guestID, userAgent)

	// Generate the XPFF header
	headerHex, err := handler.GetXPFFHeader()
	if err != nil {
		t.Fatalf("GetXPFFHeader returned error: %v", err)
	}
	if headerHex == "" {
		t.Fatal("GetXPFFHeader returned empty string")
	}

	// Decrypt it using the same encoded guestID
	encodedGuestID := url.QueryEscape(guestID)
	plaintext, err := handler.generator.DecodeXPFF(headerHex, encodedGuestID)
	if err != nil {
		t.Fatalf("DecodeXPFF returned error: %v", err)
	}

	// Unmarshal and verify the JSON content
	var content Content
	if err = json.Unmarshal([]byte(plaintext), &content); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	props := content.NavigatorProperties
	if props.HasBeenActive != "true" {
		t.Errorf("expected HasBeenActive = \"true\", got %q", props.HasBeenActive)
	}
	if props.UserAgent != userAgent {
		t.Errorf("expected UserAgent = %q, got %q", userAgent, props.UserAgent)
	}
	if props.Webdriver != "false" {
		t.Errorf("expected Webdriver = \"false\", got %q", props.Webdriver)
	}

	// CreatedAt should be very recent (within 2 seconds)
	now := time.Now().UnixMilli()
	if content.CreatedAt < now-2000 || content.CreatedAt > now+2000 {
		t.Errorf("CreatedAt = %d not within [%d, %d]", content.CreatedAt, now-2000, now+2000)
	}
}

func TestHandler_GetXPFFHeader_WrongGuestID(t *testing.T) {
	guestID := "v1:174864550766043618"
	userAgent := "test-agent/1.0"
	handler := NewHandler(guestID, userAgent)

	headerHex, err := handler.GetXPFFHeader()
	if err != nil {
		t.Fatalf("GetXPFFHeader returned error: %v", err)
	}

	// Try to decode with an incorrect guestID
	wrongGuest := url.QueryEscape("v1:000000000000000000")
	_, err = handler.generator.DecodeXPFF(headerHex, wrongGuest)
	if err == nil {
		t.Fatal("expected DecodeXPFF to error with wrong guestID, but got nil")
	}
}
