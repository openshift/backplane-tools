package rosa

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

	matches := github.FindAssetsForArchAndOS(release.Assets)
	binaryMatches := github.FindAssetsExcluding([]string{"sha256"}, matches)
	if len(binaryMatches) != 1 {
		return fmt.Errorf("unexpected number of assets found matching system spec: expected 1, got %d.\nMatching assets: %v", len(binaryMatches), binaryMatches)
	}
	rosaBinaryAsset := binaryMatches[0]

	checksumMatches := github.FindAssetsContaining([]string{"sha256"}, matches)
	if len(checksumMatches) != 1 {
		return fmt.Errorf("unexpected number of checksum assets found: expected 1, got %d.\nMatching assets: %v", len(matches), matches)
	}
	checksumAsset := checksumMatches[0]

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
