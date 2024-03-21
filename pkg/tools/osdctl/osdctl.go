package osdctl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gogithub "github.com/google/go-github/v51/github"

	"github.com/openshift/backplane-tools/pkg/sources/github"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/utils"
)

const (
	toolChecksumAssetName = "sha256sum.txt"
)

// Tool implements the interface to manage the 'osdctl' binary
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default: base.NewDefault("osdctl"),
			Source:  github.NewSource("openshift", "osdctl"),
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

	matches := github.FindAssetsForArchAndOS(release.Assets)
	if len(matches) != 1 {
		return fmt.Errorf("unexpected number of assets found matching system spec: expected 1, got %d.\nMatching assets: %v", len(matches), matches)
	}
	toolArchiveAsset := matches[0]

	matches = github.FindAssetsContaining([]string{toolChecksumAssetName}, release.Assets)
	if len(matches) != 1 {
		return fmt.Errorf("unexpected number of checksum assets found: expected 1, got %d.\nMatching assets: %v", len(matches), matches)
	}
	checksumAsset := matches[0]

	// Download the arch- & os-specific assets
	toolDir := t.ToolDir()
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.Source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, toolArchiveAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	toolArchiveFilepath := filepath.Join(versionedDir, toolArchiveAsset.GetName())
	binarySum, err := utils.Sha256sum(toolArchiveFilepath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", toolArchiveFilepath, err)
	}

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumLine, err := utils.GetLineInFileMatchingKey(checksumFilePath, toolArchiveAsset.GetName())
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file '%s': %w", checksumFilePath, err)
	}
	checksumTokens := strings.Fields(checksumLine)
	if len(checksumTokens) != 2 {
		return fmt.Errorf("the checksum file '%s' is invalid: expected 2 fields, got %d", checksumFilePath, len(checksumTokens))
	}
	actual := checksumTokens[0]

	toolExecutable := t.ExecutableName()
	if strings.TrimSpace(binarySum) != strings.TrimSpace(actual) {
		return fmt.Errorf("warning: Checksum for '%s' does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s", toolExecutable, toolArchiveAsset.GetBrowserDownloadURL())
	}

	// Untar binary bundle
	err = utils.Unarchive(toolArchiveFilepath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive the '%s' asset file '%s': %w", toolExecutable, toolArchiveFilepath, err)
	}

	// Link as latest
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing '%s' binary at '%s': %w", toolExecutable, base.LatestDir, err)
	}

	toolBinaryFilepath := filepath.Join(versionedDir, toolExecutable)
	err = os.Symlink(toolBinaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new '%s' binary to '%s': %w", toolExecutable, base.LatestDir, err)
	}
	return nil
}
