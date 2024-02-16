package ocmaddons

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

// Tool implements the interface to manage the 'ocm-addons' binary
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default: base.NewDefaultWithExecutable("ocm-addons", "ocm-addons"),
			Source:  github.NewSource("mt-sre", "ocm-addons"),
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
	ocmaddonsArchiveAsset := matches[0]

	matches = github.FindAssetsContaining([]string{"checksums.txt"}, release.Assets)
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

	err = t.Source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, ocmaddonsArchiveAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	ocmaddonsArchiveFilepath := filepath.Join(versionedDir, ocmaddonsArchiveAsset.GetName())
	binarySum, err := utils.Sha256sum(ocmaddonsArchiveFilepath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", ocmaddonsArchiveFilepath, err)
	}

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumLine, err := utils.GetLineInFileMatchingKey(checksumFilePath, ocmaddonsArchiveAsset.GetName())
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
		return fmt.Errorf("warning: Checksum for '%s' does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s", toolExecutable, ocmaddonsArchiveAsset.GetBrowserDownloadURL())
	}

	// Untar binary bundle
	err = utils.Unarchive(ocmaddonsArchiveFilepath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive the '%s' asset file '%s': %w", toolExecutable, ocmaddonsArchiveFilepath, err)
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
