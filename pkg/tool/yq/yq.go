package yq

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gogithub "github.com/google/go-github/v51/github"

	"github.com/openshift/backplane-tools/pkg/source/github"
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool implements the interface to manage the 'yq' binary
type Tool struct {
	source *github.Source

	name string
}

func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("mikefarah", "yq"),
		name:   "yq",
	}
	return t
}

func (t *Tool) Name() string {
	return t.name
}

func (t *Tool) Install(rootDir, latestDir string) error {
	// Pull latest release from GH
	release, err := t.source.FetchLatestRelease()
	if err != nil {
		return err
	}

	// Determine which assets to download
	var checksumAsset *gogithub.ReleaseAsset
	var binaryAsset *gogithub.ReleaseAsset
	for _, asset := range release.Assets {
		if asset.GetName() == "checksums" {
			if checksumAsset.GetName() != "" {
				return errors.New("detected duplicate checksum assets")
			}
			checksumAsset = asset
			continue
		}
		if strings.Contains(asset.GetName(), ".tar.gz") {
			continue
		}
		if !strings.Contains(asset.GetName(), runtime.GOARCH) {
			continue
		}
		if !strings.Contains(strings.ToLower(asset.GetName()), strings.ToLower(runtime.GOOS)) {
			continue
		}

		if binaryAsset.GetName() != "" {
			return fmt.Errorf("detected duplicate binary asset")
		}
		binaryAsset = asset
	}
	// Ensure both checksum and binary were retrieved
	if checksumAsset.GetName() == "" {
		return fmt.Errorf("failed to find checksum asset")
	}
	if binaryAsset.GetName() == "" {
		return fmt.Errorf("failed to find the binary asset")
	}

	// Download the arch- & os-specific assets
	toolDir := t.toolDir(rootDir)
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, binaryAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	binaryFilepath := filepath.Join(versionedDir, binaryAsset.GetName())
	binarySum, err := utils.Sha256sum(binaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", binaryFilepath, err)
	}

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumLine, err := utils.GetLineInFile(checksumFilePath, binaryAsset.GetName())
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file '%s': %w", checksumFilePath, err)
	}

	// For some reason, yq ships several checksum formats for each asset in its 'checksums' file.
	// Its honestly less fragile to check if _any_ of the columns contain our calculated checksum than try to decipher which column corresponds to which format
	if !strings.Contains(checksumLine, strings.TrimSpace(binarySum)) {
		return fmt.Errorf("warning: Checksum for yq does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s", binaryAsset.GetBrowserDownloadURL())
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'yq' binary at '%s': %w", latestDir, err)
	}
	err = os.Symlink(filepath.Join(versionedDir, binaryAsset.GetName()), latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'yq' binary to '%s': %w", latestDir, err)
	}
	return nil
}

func (t *Tool) Installed(rootDir string) (bool, error) {
	toolDir := t.toolDir(rootDir)
	return utils.FileExists(toolDir)
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, t.name)
}

func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, t.name)
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
