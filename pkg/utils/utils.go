package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Contains returns true if the provided list has a matching element
func Contains[T comparable](list []T, val T) bool {
	for _, elem := range list {
		if elem == val {
			return true
		}
	}
	return false
}

// Keys returns a slice containing the keys of the provided map.
// Order is preserved
func Keys[T, U comparable](myMap map[T]U) []T {
	keys := []T{}
	for k := range myMap {
		keys = append(keys, k)
	}
	return keys
}

// GetLineInFile searches the provided file for a line that contains the
// provided string. If a match is found, the entire line is returned.
// Only the first result is returned. If no lines match, an error is returned
func GetLineInFile(filepath, match string) (res string, err error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			// Override the returned error if no other error
			// is being returned
			err = closeErr
		}
	}()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, match) {
			return line, nil
		}
	}
	return "", fmt.Errorf("failed to find line matching '%s' in '%s'", match, filepath)
}
