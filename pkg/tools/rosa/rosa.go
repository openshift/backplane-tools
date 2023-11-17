package rosa

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gogithub "github.com/google/go-github/v51/github"
	"github.com/openshift/backplane-tools/pkg/sources/github"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool implements the interface to manage the 'rosa' binary
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default: base.Default{Name: "rosa"},
			Source:  github.NewSource("openshift", "rosa"),
		},
	}
	return t
}

func (t *Tool) Install() error {
	// Pull latest release from GH
	release, err := t.Source.FetchLatestRelease()
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
	toolDir := t.ToolDir()
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.Source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, rosaBinaryAsset}, versionedDir)
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
	checksumLine, err := utils.GetLineInFileMatchingKey(checksumFilePath, rosaBinaryAsset.GetName())
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file '%s': %w", checksumFilePath, err)
	}
	checksumTokens := strings.Fields(checksumLine)
	if len(checksumTokens) != 2 {
		return fmt.Errorf("the checksum file '%s' is invalid: expected 2 fields, got %d", checksumFilePath, len(checksumTokens))
	}
	actual := checksumTokens[0]

	if strings.TrimSpace(binarySum) != strings.TrimSpace(actual) {
		return fmt.Errorf("warning: Checksum for rosa does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s", rosaBinaryAsset.GetBrowserDownloadURL())
	}

	// Link as latest
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'rosa' binary at '%s': %w", base.LatestDir, err)
	}

	err = os.Symlink(rosaBinaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'rosa' binary to '%s': %w", base.LatestDir, err)
	}
	return nil
}
