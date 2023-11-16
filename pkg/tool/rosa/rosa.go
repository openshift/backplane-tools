package rosa

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gogithub "github.com/google/go-github/v51/github"
	"github.com/openshift/backplane-tools/pkg/source/github"
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool implements the interface to manage the 'rosa' binary
type Tool struct {
	source *github.Source
}

func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("openshift", "rosa"),
	}
	return t
}

func (t *Tool) Name() string {
	return "rosa"
}

func (t *Tool) Install(rootDir, latestDir string) error {
	// Pull latest release from GH
	release, err := t.source.FetchLatestRelease()
	if err != nil {
		return err
	}

	// Determine which assets to download
	var checksumAsset *gogithub.ReleaseAsset
	var rosaBinaryAsset *gogithub.ReleaseAsset
	for _, asset := range release.Assets {
		// Exclude assets that do not match system architecture
		if !strings.Contains(asset.GetName(), runtime.GOARCH) {
			continue
		}
		// Exclude assets that do not match system OS
		if !strings.Contains(strings.ToLower(asset.GetName()), strings.ToLower(runtime.GOOS)) {
			continue
		}
		if strings.Contains(asset.GetName(), "sha256") {
			if checksumAsset.GetName() != "" {
				return fmt.Errorf("detected duplicate rosa checksum assets")
			}
			checksumAsset = asset
			continue
		}

		if rosaBinaryAsset.GetName() != "" {
			return fmt.Errorf("detected duplicate rosa binary assets")
		}
		rosaBinaryAsset = asset
	}
	// Ensure both checksum and binary were retrieved
	if rosaBinaryAsset.GetName() == "" {
		return fmt.Errorf("failed to find valid rosa binary asset")
	}
	if checksumAsset.GetName() == "" {
		return fmt.Errorf("failed to find valid rosa checksum asset")
	}

	// Download the arch- & os-specific assets
	toolDir := t.toolDir(rootDir)
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, rosaBinaryAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	rosaBinaryFilepath := filepath.Join(versionedDir, rosaBinaryAsset.GetName())
	binarySum, err := utils.Sha256sum(rosaBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", rosaBinaryFilepath, err)
	}

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumLine, err := utils.GetLineInFile(checksumFilePath, rosaBinaryAsset.GetName())
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file '%s': %w", checksumFilePath, err)
	}
	checksumTokens := strings.Fields(checksumLine)
	if len(checksumTokens) != 2 {
		return fmt.Errorf("the checksum file '%s' is invalid: expected 2 fields, got %d", checksumFilePath, len(checksumTokens))
	}
	actual := checksumTokens[0]

	if strings.TrimSpace(binarySum) != strings.TrimSpace(actual) {
		return fmt.Errorf("WARNING: Checksum for rosa does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s\n", rosaBinaryAsset.GetBrowserDownloadURL())
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'rosa' binary at '%s': %w", latestDir, err)
	}

	err = os.Symlink(rosaBinaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'rosa' binary to '%s': %w", latestDir, err)
	}
	return nil
}

func (t *Tool) Installed(rootDir string) (bool, error) {
	toolDir := t.toolDir(rootDir)
	return utils.FileExists(toolDir)
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "rosa")
}

// symlinkPath returns the path to the symlink created by this tool, given the latest directory
func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "rosa")
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
