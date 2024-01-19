package aws

import (
	"net/http"
	"path/filepath"

	"github.com/openshift/backplane-tools/pkg/utils"
)

func DownloadAWSCLIRelease(url string, fileExtension string, dir string) error {
	// Make the HTTP request to download the release
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Create the output file
	filePath := filepath.Join(dir, "aws-cli"+fileExtension)
	return utils.WriteFile(response.Body, filePath, 0o755)
}
