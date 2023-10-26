package golangcilint

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

// Tool implements the interface to manage the 'golangcilint' binary
type Tool struct {
	source *github.Source

	name string
}

func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("golangci", "golangci-lint"),
		name: "golangci-lint",
	}
	return t
}

func (t *Tool) Name() string {
	return "golangci-lint"
}

func (t *Tool) Install(rootDir, latestDir string) error {
	// Pull latest release from GH
	release, err := t.source.FetchLatestRelease()
	if err != nil {
		return err
	}

	// Determine which assets to download
	var checksumAsset *gogithub.ReleaseAsset
	var archiveAsset *gogithub.ReleaseAsset
	for _, asset := range release.Assets {
		if strings.Contains(asset.GetName(), "checksums.txt") {
			if checksumAsset.GetName() != "" {
				return fmt.Errorf("detected duplicate checksum assets")
			}
			checksumAsset = asset
			continue
		}

		// Exclude .deb and .rpm assets
		if !strings.Contains(asset.GetName(), ".tar.gz") {
			continue
		}

		// Exclude assets that do not match system architecture
		if !strings.Contains(asset.GetName(), runtime.GOARCH) {
			continue
		}
		// Exclude assets that do not match system OS
		if !strings.Contains(strings.ToLower(asset.GetName()), strings.ToLower(runtime.GOOS)) {
			continue
		}
		if archiveAsset.GetName() != "" {
			return fmt.Errorf("detected duplicate archive assets")
		}
		archiveAsset = asset
	}
	// Ensure both checksum and binary were retrieved
	if archiveAsset.GetName() == "" {
		return fmt.Errorf("failed to find valid archive asset")
	}
	if checksumAsset.GetName() == "" {
		return fmt.Errorf("failed to find valid checksum asset")
	}

	// Download the arch- & os-specific assets
	toolDir := t.toolDir(rootDir)
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, archiveAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	archiveFilepath := filepath.Join(versionedDir, archiveAsset.GetName())
	archiveSum, err := utils.Sha256sum(archiveFilepath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", archiveFilepath, err)
	}

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumLine, err := utils.GetLineInFile(checksumFilePath, archiveAsset.GetName())
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file '%s': %w", checksumFilePath, err)
	}
	checksumTokens := strings.Fields(checksumLine)
	if len(checksumTokens) != 2 {
			return fmt.Errorf("the checksum file '%s' is invalid: expected 2 fields, got %d", checksumFilePath, len(checksumTokens))
	}
	actual := checksumTokens[0]

	if strings.TrimSpace(archiveSum) != strings.TrimSpace(actual) {
		return fmt.Errorf("WARNING: Checksum for %s does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s\n", t.name, archiveAsset.GetBrowserDownloadURL())
	}

	// Unarchive tool
	fmt.Printf("archiveFilepath: %v\n", archiveFilepath)
	fmt.Printf("versionedDir: %v\n", versionedDir)
	err = utils.Unarchive(archiveFilepath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive %s: %w", archiveFilepath, err)
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing tool at '%s': %w", latestDir, err)
	}

	// golangci-lint's release archives contain a subdirectory that holds the binary
	archiveFile := filepath.Base(archiveFilepath)
	releaseName := strings.TrimSuffix(archiveFile, ".tar.gz")

	binaryFilepath := filepath.Join(versionedDir, releaseName, "golangci-lint")
	err = os.Symlink(binaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new binary to '%s': %w", latestDir, err)
	}
	return nil
}

func (t *Tool) Installed(rootDir string) (bool, error) {
	toolDir := t.toolDir(rootDir)
	return utils.FileExists(toolDir)
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "golangci-lint")
}

// symlinkPath returns the path to the symlink created by this tool, given the latest directory
func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "golangci-lint")
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
