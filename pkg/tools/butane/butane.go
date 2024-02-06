package butane

import (
	"fmt"
	"os"
	"path/filepath"

	gogithub "github.com/google/go-github/v51/github"

	"github.com/openshift/backplane-tools/pkg/sources/github"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool implements the interface to manage the 'butane' executable
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default: base.NewDefault("butane"),
			Source:  github.NewSource("coreos", "butane"),
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

	matches := github.FindAssetsExcluding([]string{".asc"}, github.FindAssetsForArchAndOS(release.Assets))
	if len(matches) != 1 {
		return fmt.Errorf("unexpected number of executable assets found matching system spec: expected 1, got %d.\nMatching assets: %v", len(matches), matches)
	}
	executableAsset := matches[0]

	matches = github.FindAssetsContaining([]string{".asc"}, github.FindAssetsForArchAndOS(release.Assets))
	if len(matches) != 1 {
		return fmt.Errorf("unexpected number of checksum assets found: expected 1, got %d.\nMatching assets: %v", len(matches), matches)
	}
	signatureAsset := matches[0]

	// Download the arch- & os-specific assets
	toolDir := t.ToolDir()
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.Source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{signatureAsset, executableAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify signature of downloaded assets
	executableFilepath := filepath.Join(versionedDir, executableAsset.GetName())
	signatureFilepath := filepath.Join(versionedDir, signatureAsset.GetName())

	err = utils.VerifyGPGSignature(executableFilepath, signatureFilepath)
	if err != nil {
		return fmt.Errorf("failed to verify executable signature: %w", err)
	}

	// Link as latest
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing executable at '%s': %w", base.LatestDir, err)
	}
	err = os.Symlink(executableFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new executable to '%s': %w", base.LatestDir, err)
	}
	return nil
}
