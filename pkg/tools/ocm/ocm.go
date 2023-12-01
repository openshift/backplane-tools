package ocm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gogithub "github.com/google/go-github/v51/github"
	"github.com/openshift/backplane-tools/pkg/sources/github"
	"github.com/openshift/backplane-tools/pkg/tools/base"
)

// Tool implements the interface to manage the 'ocm-cli' binary
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default: base.Default{Name: "ocm"},
			Source:  github.NewSource("openshift-online", "ocm-cli"),
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
	var ocmBinaryAsset *gogithub.ReleaseAsset
	for _, asset := range release.Assets {
		// Exclude assets that do not match system OS
		if !strings.Contains(asset.GetName(), runtime.GOOS) {
			continue
		}
		// Exclude assets that do not match system architecture
		if !strings.Contains(asset.GetName(), runtime.GOARCH) {
			continue
		}

		if strings.Contains(asset.GetName(), "sha256") {
			if checksumAsset.GetName() != "" {
				return fmt.Errorf("detected duplicate ocm-cli checksum assets")
			}
			checksumAsset = asset
			continue
		}
		if ocmBinaryAsset.GetName() != "" {
			return fmt.Errorf("detected duplicate ocm-cli binary assets")
		}
		ocmBinaryAsset = asset
	}
	// Ensure both checksum and binary were retrieved
	if checksumAsset.GetName() == "" || ocmBinaryAsset.GetName() == "" {
		return fmt.Errorf("failed to find ocm-cli or it's checksum")
	}

	// Download the arch- & os-specific assets
	toolDir := t.ToolDir()
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.Source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, ocmBinaryAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	ocmBinaryFilepath := filepath.Join(versionedDir, ocmBinaryAsset.GetName())
	fileBytes, err := os.ReadFile(ocmBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to read ocm-cli binary file '%s' while generating sha256sum: %w", ocmBinaryFilepath, err)
	}
	sumBytes := sha256.Sum256(fileBytes)
	binarySum := hex.EncodeToString(sumBytes[:])

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumBytes, err := os.ReadFile(checksumFilePath)
	if err != nil {
		return fmt.Errorf("failed to read ocm-cli checksum file '%s': %w", checksumFilePath, err)
	}
	checksum := strings.Split(string(checksumBytes), " ")[0]
	if strings.TrimSpace(binarySum) != strings.TrimSpace(checksum) {
		fmt.Printf("WARNING: Checksum for ocm-cli does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s\n", ocmBinaryAsset.GetBrowserDownloadURL())
		// We shouldn't link this binary to latest if the checksum isn't valid
		return nil
	}

	// Link as latest
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'ocm' binary at '%s': %w", base.LatestDir, err)
	}

	err = os.Symlink(ocmBinaryFilepath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'ocm' binary to '%s': %w", base.LatestDir, err)
	}
	return nil
}
