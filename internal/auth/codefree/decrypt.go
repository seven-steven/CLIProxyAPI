package codefree

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"regexp"
)

// AES-192-CBC decryption parameters (extracted from codefree-cli source code)
const decryptionKey = "Xtpa6sS&+D.NAo%CP8LA:7pk"
const decryptionIV = "%1KJIrl3!XUxr04V"

// DecryptAPIKey decrypts the encrypted API key from codefree-cli
// Returns the decrypted UUID format API key
// e.g., "3983ce76-288d-4725-9a5a-0fee50477244"
// encryptedApiKey: Base64 encoded ciphertext (48 bytes, no embedded IV)
func DecryptAPIKey(encryptedApiKey string) (string, error) {
	// Base64 decode
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedApiKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(ciphertext) != 48 {
		return "", fmt.Errorf("invalid encrypted data length: %d, expected 48", len(ciphertext))
	}

	// Decrypt using AES-192-CBC with fixed IV
	block, err := aes.NewCipher([]byte(decryptionKey))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	mode := cipher.NewCBCDecrypter(block, []byte(decryptionIV))
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// Remove PKCS#7 padding
	paddingLen := int(decrypted[len(decrypted)-1])
	if paddingLen <= 0 || paddingLen > 16 || paddingLen > len(decrypted) {
		return "", fmt.Errorf("invalid PKCS#7 padding")
	}
	decrypted = decrypted[:len(decrypted)-paddingLen]

	// Validate UUID format
	if !ValidateAPIKeyFormat(string(decrypted)) {
		return "", fmt.Errorf("decrypted data is not in valid UUID format")
	}

	return string(decrypted), nil
}

// uuidPattern is the compiled regex for UUID format validation
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidateAPIKeyFormat validates if the given string is a valid UUID format
func ValidateAPIKeyFormat(apiKey string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return uuidPattern.MatchString(apiKey)
}
