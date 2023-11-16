package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// Checksum reads the file at the provided path and calculates the sha256sum
func Sha256sum(filepath string) (string, error) {
	fileBytes, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s' while generating sha256sum: %w", filepath, err)
	}
	sumBytes := sha256.Sum256(fileBytes)
	return hex.EncodeToString(sumBytes[:]), nil
}
