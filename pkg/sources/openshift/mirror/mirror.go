/*
mirror provides the capability for tools to retrieve files from mirror.openshift.com
*/
package mirror

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	defaultBaseURL string = "http://mirror.openshift.com"
)

// Source objects retrieve files from a mirror server
type Source struct {
	// baseURL represents the url that the Source's requests should be built off of
	BaseURL string
}

// NewSource creates a Source
func NewSource() *Source {
	s := &Source{
		BaseURL: defaultBaseURL,
	}
	return s
}

// downloadFile retrieves the file from the source given the path the file
// is located at on the server and the local directory the file should be stored in
func (s Source) DownloadFile(path, dir string) (string, error) {
	url, err := s.BuildURL(path)
	if err != nil {
		return "", fmt.Errorf("failed to build URL: %w", err)
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to GET '%s': %w", url, err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			fmt.Printf("WARNING: failed to close response body: %v\n", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-%d status code: %d", http.StatusOK, resp.StatusCode)
	}

	_, fileName := filepath.Split(path)
	filePath := filepath.Join(dir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file '%s': %w", filePath, err)
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			fmt.Printf("warning: failed to close %s\n", file.Name())
		}
	}()

	_, err = file.ReadFrom(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to download the contents of '%s' into '%s': %w", url, file.Name(), err)
	}

	err = file.Sync()
	if err != nil {
		return "", fmt.Errorf("failed to sync contents of '%s' to disk: %w", file.Name(), err)
	}
	return file.Name(), nil
}

// GetFileContents returns the contents of the specified file without storing it locally.
// It is the callers responsibility to Close() the file after reading
func (s Source) GetFileContents(path string) (io.ReadCloser, error) {
	url, err := s.BuildURL(path)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to GET '%s': %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-%d status code: %d", http.StatusOK, resp.StatusCode)
	}
	return resp.Body, nil
}

// buildURL constructs the full URL the source should operate on given the path of the file we're trying to retrieve
func (s Source) BuildURL(path string) (string, error) {
	return url.JoinPath(s.BaseURL, path)
}
