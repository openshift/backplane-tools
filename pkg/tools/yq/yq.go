package yq

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

// Tool implements the interface to manage the 'yq' binary
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default: base.NewDefault("yq"),
			Source:  github.NewSource("mikefarah", "yq"),
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

	matches := github.FindAssetsExcluding([]string{".tar.gz"}, github.FindAssetsForArchAndOS(release.Assets))
	if len(matches) != 1 {
		return fmt.Errorf("unexpected number of assets found matching system spec: expected 1, got %d.\nMatching assets: %v", len(matches), matches)
	}
	toolAsset := matches[0]

	matches, err = github.FindAssetsMatching("^checksums$", release.Assets)
	if err != nil {
		return fmt.Errorf("failed to filter assets by regular expression: %w", err)
	}
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

	err = t.Source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, toolAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	toolArchiveFilepath := filepath.Join(versionedDir, toolAsset.GetName())
	binarySum, err := utils.Sha256sum(toolArchiveFilepath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", toolArchiveFilepath, err)
	}

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumLine, err := utils.GetLineInFileMatchingKey(checksumFilePath, toolAsset.GetName())
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file '%s': %w", checksumFilePath, err)
	}

	// For some reason, yq ships several checksum formats for each asset in its 'checksums' file.
	// Its honestly less fragile to check if _any_ of the columns contain our calculated checksum than try to decipher which column corresponds to which format
	if !strings.Contains(checksumLine, strings.TrimSpace(binarySum)) {
		return fmt.Errorf("warning: Checksum for yq does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s", toolAsset.GetBrowserDownloadURL())
	}

	// Link as latest
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing '%s' binary at '%s': %w", toolAsset.GetName(), base.LatestDir, err)
	}

	toolBinaryFilepath := filepath.Join(versionedDir, toolAsset.GetName())
	err = os.Symlink(toolBinaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new '%s' binary to '%s': %w", toolAsset.GetName(), base.LatestDir, err)
	}
	return nil
}
