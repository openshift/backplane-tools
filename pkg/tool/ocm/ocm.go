package ocm

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gogithub "github.com/google/go-github/v51/github"
	"github.com/openshift/backplane-tools/pkg/source/github"
)

// Tool implements the interface to manage the 'ocm-cli' binary
type Tool struct {
	source *github.Source
}

// NewTool builds an OCM tool object
func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("openshift-online", "ocm-cli"),
	}
	return t
}

// Name returns the name this tool should be referenced by
func (t *Tool) Name() string {
	return "ocm"
}

// Install downloads this tool, verifies it's integrity via checksum, and links the newly downloaded binary
// to the "latest" directory
func (t *Tool) Install(rootDir, latestDir string) error {
	// Pull latest release from GH
	release, err := t.source.FetchLatestRelease()
	if err != nil {
		return err
	}

	// Determine which assets to download
	var checksumAsset *gogithub.ReleaseAsset
	var ocmBinaryAsset *gogithub.ReleaseAsset
	for _, asset := range release.Assets {
		// Exclude assets that do not match system OS
		if !strings.Contains(asset.GetName(), runtime.GOOS) {
			continue
		}
		// Exclude assets that do not match system architecture
		if !strings.Contains(asset.GetName(), runtime.GOARCH) {
			continue
		}

		if strings.Contains(asset.GetName(), "sha256") {
			if checksumAsset.GetName() != "" {
				return fmt.Errorf("detected duplicate ocm-cli checksum assets")
			}
			checksumAsset = asset
			continue
		}
		if ocmBinaryAsset.GetName() != "" {
			return fmt.Errorf("detected duplicate ocm-cli binary assets")
		}
		ocmBinaryAsset = asset
	}
	// Ensure both checksum and binary were retrieved
	if ocmBinaryAsset.GetName() == "" {
		return fmt.Errorf("failed to find a valid ocm binary for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if checksumAsset.GetName() == "" {
		return fmt.Errorf("failed to find a valid checksum file for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download the arch- & os-specific assets
	toolDir := t.toolDir(rootDir)
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, ocmBinaryAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	ocmBinaryFilepath := filepath.Join(versionedDir, ocmBinaryAsset.GetName())
	fileBytes, err := os.ReadFile(ocmBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to read ocm-cli binary file '%s' while generating sha256sum: %w", ocmBinaryFilepath, err)
	}
	sumBytes := sha256.Sum256(fileBytes)
	// TODO - there's probably a better way to do this
	binarySum := fmt.Sprintf("%x", sumBytes[:])

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumBytes, err := os.ReadFile(checksumFilePath)
	if err != nil {
		return fmt.Errorf("failed to read ocm-cli checksum file '%s': %w", checksumFilePath, err)
	}
	tokens := strings.Fields(strings.TrimSpace(string(checksumBytes)))
	if len(tokens) != 2 {
		return fmt.Errorf("invalid checksum file: expected 2 tokens, got %d", len(tokens))
	}
	if strings.TrimSpace(binarySum) != strings.TrimSpace(tokens[0]) {
		return fmt.Errorf("the provided checksum for ocm-cli does not match the calculated value. Please retry installation. If issues persists, this tool can be downloaded manually at %s\n", ocmBinaryAsset.GetBrowserDownloadURL())
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'ocm' binary at '%s': %w", latestDir, err)
	}

	err = os.Symlink(ocmBinaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'ocm' binary to '%s': %w", latestDir, err)
	}
	return nil
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "ocm")
}

// symlinkPath returns the path to the symlink which points to the application's latest version, given
// the "latest" directory where this link should exist
func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "ocm")
}

// Remove cleans any tool files from the provided directories
func (t *Tool) Remove(rootDir, latestDir string) error {
	// Remove all binaries owned by this tool
	toolDir := t.toolDir(rootDir)
	err := os.RemoveAll(toolDir)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", toolDir, err)
	}

	// Remove all symlinks owned by this tool
	latestFilePath := t.symlinkPath(latestDir)
	err = os.RemoveAll(latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to remove symlinked file %s: %w", latestFilePath, err)
	}
	return nil
}
