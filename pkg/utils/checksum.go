package utils

import (
	"crypto/sha256"
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
	// TODO - there's probably a better way to do this
	return fmt.Sprintf("%x", sumBytes[:]), nil

}
