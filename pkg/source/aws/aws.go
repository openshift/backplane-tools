package aws

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadAWSCLIRelease(version string, dir string) error {
	url := "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-" + version + ".zip"

	// Create the output file
	filePath := filepath.Join(dir, "aws-cli.zip")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	// Make the HTTP request to download the release
	response, err := http.Get(url)
	if err != nil {
		return err
	}

	// Write the response body to the output file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}
