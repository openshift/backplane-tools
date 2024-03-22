package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
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

// ContainsAll returns true if all of the provided search terms are a substring of 'str'
func ContainsAll(str string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(str, term) {
			return false
		}
	}
	return true
}

// ContainsAny returns true if any of the provided search terms are a substring of 'str'
func ContainsAny(str string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(str, term) {
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

// FileExists checks if a file *of any type* is present at the given path
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetLineInFileMatchingKey searches the provided file for a line that contains the
// provided key. A key is a pattern that will be either at the begin/end of line and
// will have ::spaces:: characters around.
// If a match is found, the entire line is returned.
// Only the first result is returned. If no lines match, an error is returned
func GetLineInFileMatchingKey(filepath string, key string) (res string, err error) {
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

	r, err := regexp.Compile("(^|\\s)" + key + "($|\\s)")
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Err() != nil {
			return "", fmt.Errorf("failed to read line: %w", err)
		}
		line := scanner.Text()

		match := r.FindStringSubmatch(line)
		if len(match) > 0 {
			return line, nil
		}
	}

	return "", errors.New("no match found")
}

// GetLinInReader searches the provided reader for a line that contains the
// provided string. If a match is found, the entire line is returned.
// Only the first result is returned. If no lines match, an error is returned
func GetLineInReader(reader io.Reader, match string) (res string, err error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if scanner.Err() != nil {
			return "", fmt.Errorf("failed to read line: %w", err)
		}
		line := scanner.Text()

		if strings.Contains(line, match) {
			return line, nil
		}
	}
	return "", fmt.Errorf("failed to find matching line for search pattern: '%s'", match)
}

// WriteFile creates a new file using the contents provided ('from') at the path provided ('to')
func WriteFile(from io.Reader, to string, permissions os.FileMode) error {
	file, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("failed to create file '%s': %w", to, err)
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to close file '%s': %v", to, closeErr)
		}
	}()

	err = file.Chmod(permissions)
	if err != nil {
		return fmt.Errorf("failed to set permissions on file '%s': %w", to, err)
	}

	_, err = file.ReadFrom(from)
	if err != nil {
		return fmt.Errorf("failed to write to file '%s': %w", to, err)
	}

	return nil
}

// GetArchAliases returns all commonly used names for the system's architecture.
// ie - An 'amd64' system is functionally equivalent to 'x86_64' for our purposes
func GetArchAliases() []string {
	switch runtime.GOARCH {
	case "amd64":
		return []string{"amd64", "x86_64"}
	default:
		return []string{runtime.GOARCH}
	}
}

// GetOSAliases returns all commonly used names for the system's OS.
// ie - A system running 'darwin' is functionally equivalent to 'mac' for our purposes
func GetOSAliases() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"darwin", "mac"}
	default:
		return []string{runtime.GOOS}
	}
}
