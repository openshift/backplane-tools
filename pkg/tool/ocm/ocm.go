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
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool implements the interface to manage the 'ocm-cli' binary
type Tool struct {
	source *github.Source
}

func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("openshift-online", "ocm-cli"),
	}
	return t
}

func (t *Tool) Name() string {
	return "ocm"
}

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
	if checksumAsset.GetName() == "" || ocmBinaryAsset.GetName() == "" {
		return fmt.Errorf("failed to find ocm-cli or it's checksum")
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
	checksum := strings.Split(string(checksumBytes), " ")[0]
	if strings.TrimSpace(binarySum) != strings.TrimSpace(checksum) {
		fmt.Printf("WARNING: Checksum for ocm-cli does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s\n", ocmBinaryAsset.GetBrowserDownloadURL())
		// We shouldn't link this binary to latest if the checksum isn't valid
		return nil
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

func (t *Tool) Installed(rootDir string) (bool, error) {
	toolDir := t.toolDir(rootDir)
	return utils.FileExists(toolDir)
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "ocm")
}

func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "ocm")
}

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
