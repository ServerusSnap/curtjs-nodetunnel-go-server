package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

// Вернёт ID в виде строки из 8 символом
func GenerateOID() string {
	randomBytes := make([]byte, 4)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatalf("Error generating random bytes: %v", err)
	}

	shortID := hex.EncodeToString(randomBytes)
	return shortID
}
