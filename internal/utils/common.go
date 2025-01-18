package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
)

// GenerateUniqueID creates a unique identifier.
func GenerateUniqueID() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 128 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}
	return hex.EncodeToString(bytes), nil
}

// logShardResponse logs the response details from the shard API.
func LogResponse(resp *http.Response) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read  response body: %v", err)
		return
	}

	log.Printf("Response: Status Code: %d, Body: %s", resp.StatusCode, string(body))
}
