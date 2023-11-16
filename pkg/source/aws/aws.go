package aws

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadAWSCLIRelease(url string, fileExtension string, dir string) error {
	// Create the output file
	filePath := filepath.Join(dir, "aws-cli"+fileExtension)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	// Make the HTTP request to download the release
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Write the response body to the output file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}
