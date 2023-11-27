package oc

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	openshiftmirror "github.com/openshift/backplane-tools/pkg/source/openshift/mirror"
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool implements the interface to manage the 'backplane-cli' binary
type Tool struct {
	source *openshiftmirror.Source
}

func NewTool() *Tool {
	t := &Tool{
		source: openshiftmirror.NewSource(),
	}
	return t
}

func (t *Tool) Name() string {
	return "oc"
}

func (t *Tool) Install(rootDir, latestDir string) error {
	baseSlug := fmt.Sprintf("/pub/openshift-v4/%s/clients/ocp/stable/", runtime.GOARCH)

	// Retrieve latest release info to determine which version we're operating on
	ocReleaseSlug := fmt.Sprintf("%s/release.txt", baseSlug)
	version, err := t.getVersion(ocReleaseSlug)
	if err != nil {
		return fmt.Errorf("failed to retrieve version info: %w", err)
	}

	versionedDir := filepath.Join(t.toolDir(rootDir), version)
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	// Download client archive
	clientArchiveName := fmt.Sprintf("openshift-client-%s-%s.tar.gz", runtime.GOOS, version)
	if runtime.GOOS == "darwin" {
		// 'darwin' OSes are referred to as 'mac' in mirror.openshift.com
		clientArchiveName = fmt.Sprintf("openshift-client-mac-%s.tar.gz", version)
	}

	clientArchiveSlug, err := url.JoinPath(baseSlug, clientArchiveName)
	if err != nil {
		return fmt.Errorf("failed to build client URL: %w", err)
	}
	clientArchiveFilePath, err := t.source.DownloadFile(clientArchiveSlug, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download client archive file %s: %w", clientArchiveSlug, err)
	}
	err = os.Chmod(clientArchiveFilePath, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to update file mode for %s: %w", clientArchiveFilePath, err)
	}

	// Download latest checksum file
	checksumSlug, err := url.JoinPath(baseSlug, "sha256sum.txt")
	if err != nil {
		return fmt.Errorf("failed to build checksum URL: %w", err)
	}

	checksumFilePath, err := t.source.DownloadFile(checksumSlug, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download checksum file %s: %w", checksumSlug, err)
	}

	checksum, err := t.extractChecksumFromFile(checksumFilePath, clientArchiveName)
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file %s: %w", checksumFilePath, err)
	}

	// Checksum client archive & compare
	archiveSum, err := utils.Sha256sum(clientArchiveFilePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", clientArchiveFilePath, err)
	}

	if strings.TrimSpace(archiveSum) != strings.TrimSpace(checksum) {
		sourceURL, err := t.source.BuildURL(clientArchiveSlug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct source URL for manual retrieval: %v\n", err)
			return fmt.Errorf("checksum for %s does not match the calculated value: expected '%s', got '%s'. Please retry installation", clientArchiveFilePath, strings.TrimSpace(checksum), strings.TrimSpace(archiveSum))
		}
		return fmt.Errorf("checksum for %s does not match the calculated value: expected '%s', got '%s'. Please retry installation. If issue persists, this tool can be downloaded manually at %s", clientArchiveFilePath, strings.TrimSpace(checksum), strings.TrimSpace(archiveSum), sourceURL)
	}

	// Unarchive client
	err = utils.Unarchive(clientArchiveFilePath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive %s: %w", clientArchiveFilePath, err)
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing symlink at '%s': %w", latestFilePath, err)
	}

	clientBinaryFilepath := filepath.Join(versionedDir, "oc")
	err = os.Symlink(clientBinaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link %s to %s: %w", clientBinaryFilepath, latestFilePath, err)
	}

	return nil
}

func (t *Tool) Installed(rootDir string) (bool, error) {
	toolDir := t.toolDir(rootDir)
	return utils.FileExists(toolDir)
}

// getVersion retrieves the version info contained within the provided release.txt file
func (t Tool) getVersion(releaseSlug string) (string, error) {
	releaseData, err := t.source.GetFileContents(releaseSlug)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve release info from %s: %w", releaseSlug, err)
	}
	defer func() {
		closeErr := releaseData.Close()
		if closeErr != nil {
			fmt.Printf("WARNING: failed to close response body: %v\n", closeErr)
		}
	}()

	line, err := utils.GetLineInReader(releaseData, "Version:")
	if err != nil {
		return "", fmt.Errorf("failed to determine version info from release file: %w", err)
	}

	tokens := strings.Fields(line)
	if len(tokens) != 2 {
		return "", fmt.Errorf("failed to parse version info from release: expected 2 tokens, got %d.\nVersion info retrieved:\n%s", len(tokens), line)
	}
	if tokens[0] != "Version:" {
		return "", fmt.Errorf("failed to parse version info from release: expected line to begin with 'Version:', got '%s'.\nVersion info retrieved:\n%s", tokens[0], line)
	}
	return tokens[1], nil
}

func (t Tool) extractChecksumFromFile(checksumFile, searchPattern string) (string, error) {
	line, err := utils.GetLineInFileMatchingKey(checksumFile, searchPattern)
	if err != nil {
		return "", err
	}

	tokens := strings.Fields(line)
	if len(tokens) != 2 {
		return "", fmt.Errorf("failed to parse checksum info: expected 2 tokens, got %d.\nChecksum info retrieved:\n%s", len(tokens), line)
	}
	return tokens[0], nil
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "oc")
}

// symlinkPath returns the path to the symlink created by this tool, given the latest directory
func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "oc")
}

// Remove completely removes this tool from the provided locations
func (t *Tool) Remove(rootDir, latestDir string) error {
	// Remove all binaries owned by this tool
	toolDir := t.toolDir(rootDir)
	err := os.RemoveAll(toolDir)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", toolDir, err)
	}

	// Remove all symlinks owned by this tool
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to remove symlinked file %s: %w", latestFilePath, err)
	}
	return nil
}

func (t *Tool) Configure() error {
	return nil
}
