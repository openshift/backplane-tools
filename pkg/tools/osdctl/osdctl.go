package osdctl

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

// Tool implements the interface to manage the 'osdctl' binary
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default: base.Default{Name: "osdctl"},
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

	// Determine which assets to download
	var checksumAsset *gogithub.ReleaseAsset
	var osdctlBinaryAsset *gogithub.ReleaseAsset
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	for _, asset := range release.Assets {
		if asset.GetName() == "sha256sum.txt" {
			if checksumAsset.GetName() != "" {
				return fmt.Errorf("detected duplicate osdctl checksum assets")
			}
			checksumAsset = asset
			continue
		}
		if !strings.Contains(asset.GetName(), arch) {
			continue
		}
		if !strings.Contains(strings.ToLower(asset.GetName()), strings.ToLower(runtime.GOOS)) {
			continue
		}

		if osdctlBinaryAsset.GetName() != "" {
			return fmt.Errorf("detected duplicate osdctl binary asset")
		}
		osdctlBinaryAsset = asset
	}
	// Ensure both checksum and binary were retrieved
	if checksumAsset.GetName() == "" || osdctlBinaryAsset.GetName() == "" {
		return fmt.Errorf("failed to find osdctl or it's checksum")
	}

	// Download the arch- & os-specific assets
	toolDir := t.ToolDir()
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.Source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, osdctlBinaryAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	osdctlBinaryFilepath := filepath.Join(versionedDir, osdctlBinaryAsset.GetName())
	binarySum, err := utils.Sha256sum(osdctlBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for '%s': %w", osdctlBinaryFilepath, err)
	}

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumLine, err := utils.GetLineInFileMatchingKey(checksumFilePath, osdctlBinaryAsset.GetName())
	if err != nil {
		return fmt.Errorf("failed to retrieve checksum from file '%s': %w", checksumFilePath, err)
	}
	checksumTokens := strings.Fields(checksumLine)
	if len(checksumTokens) != 2 {
		return fmt.Errorf("the checksum file '%s' is invalid: expected 2 fields, got %d", checksumFilePath, len(checksumTokens))
	}
	actual := checksumTokens[0]

	if strings.TrimSpace(binarySum) != strings.TrimSpace(actual) {
		return fmt.Errorf("warning: Checksum for osdctl does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s", osdctlBinaryAsset.GetBrowserDownloadURL())
	}

	// Untar osdctl file
	err = utils.Unarchive(osdctlBinaryFilepath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive the osdctl asset file '%s': %w", osdctlBinaryFilepath, err)
	}

	// Link as latest
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'osdctl' binary at '%s': %w", base.LatestDir, err)
	}
	err = os.Symlink(filepath.Join(versionedDir, "osdctl"), latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'osdctl' binary to '%s': %w", base.LatestDir, err)
	}
	return nil
}
