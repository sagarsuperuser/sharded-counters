package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateUniqueID creates a unique identifier.
func GenerateUniqueID() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 128 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}
	return hex.EncodeToString(bytes), nil
}
