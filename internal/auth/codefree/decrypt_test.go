package codefree

import (
	"testing"
)

func TestDecryptAPIKey(t *testing.T) {
	// Test data from codefree-cli credentials
	encrypted := "0Wt6fhOXuNpD9JJ6c94Xw+B95JaflyXkPqOWJlQ/9xI4fA9PwM/YgUyHDzZfW9ru"
	expected := "3983ce76-288d-4725-9a5a-0fee50477244"

	decrypted, err := DecryptAPIKey(encrypted)
	if err != nil {
		t.Fatalf("DecryptAPIKey failed: %v", err)
	}

	if decrypted != expected {
		t.Errorf("Expected %q, got %q", expected, decrypted)
	}
}

func TestDecryptAPIKey_InvalidBase64(t *testing.T) {
	_, err := DecryptAPIKey("not-valid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}
}

func TestDecryptAPIKey_InvalidLength(t *testing.T) {
	// Valid base64 but wrong length
	_, err := DecryptAPIKey("YWJjZA==") // "abcd" in base64
	if err == nil {
		t.Error("Expected error for invalid length, got nil")
	}
}

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		apiKey   string
		expected bool
	}{
		{"3983ce76-288d-4725-9a5a-0fee50477244", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"ffffffff-ffff-ffff-ffff-ffffffffffff", true},
		{"3983CE76-288D-4725-9A5A-0FEE50477244", false}, // uppercase not valid
		{"3983ce76-288d-4725-9a5a-0fee5047724", false},  // too short
		{"3983ce76-288d-4725-9a5a-0fee504772444", false}, // too long
		{"not-a-uuid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := ValidateAPIKeyFormat(tt.apiKey)
		if result != tt.expected {
			t.Errorf("ValidateAPIKeyFormat(%q) = %v, expected %v", tt.apiKey, result, tt.expected)
		}
	}
}
