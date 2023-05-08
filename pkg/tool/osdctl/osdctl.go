package osdctl

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gogithub "github.com/google/go-github/v51/github"

	"github.com/openshift/backplane-tools/pkg/source/github"
)

// Tool implements the interface to manage the 'osdctl' binary
type Tool struct {
	source *github.Source
}

func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("openshift", "osdctl"),
	}
	return t
}

func (t *Tool) Name() string {
	return "osdctl"
}

func (t *Tool) Install(rootDir, latestDir string) error {
	// Pull latest release from GH
	release, err := t.source.FetchLatestRelease()
	if err != nil {
		return err
	}

	// Determine which assets to download
	var checksumAsset *gogithub.ReleaseAsset
	var osdctlBinaryAsset *gogithub.ReleaseAsset
	var arch = runtime.GOARCH
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
	toolDir := t.toolDir(rootDir)
	versionedDir := filepath.Join(toolDir, release.GetTagName())
	err = os.MkdirAll(versionedDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = t.source.DownloadReleaseAssets([]*gogithub.ReleaseAsset{checksumAsset, osdctlBinaryAsset}, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download one or more assets: %w", err)
	}

	// Verify checksum of downloaded assets
	osdctlBinaryFilepath := filepath.Join(versionedDir, osdctlBinaryAsset.GetName())
	fileBytes, err := os.ReadFile(osdctlBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to read osdctl binary file '%s' while generating sha256sum: %w", osdctlBinaryFilepath, err)
	}
	sumBytes := sha256.Sum256(fileBytes)
	// TODO - there's probably a better way to do this
	binarySum := fmt.Sprintf("%x", sumBytes[:])

	checksumFilePath := filepath.Join(versionedDir, checksumAsset.GetName())
	checksumFile, err := os.Open(checksumFilePath)
	if err != nil {
		return fmt.Errorf("failed to open osdctl checksum file '%s': %w", checksumFilePath, err)
	}
	var checksum string
	scanner := bufio.NewScanner(checksumFile)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) != 2 {
			return fmt.Errorf("the checksum file '%s' is invalid", checksumFile.Name())
		}

		if osdctlBinaryAsset.GetName() != tokens[1] {
			continue
		}
		checksum = tokens[0]
	}
	err = checksumFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close checksumfile")
	}

	if strings.TrimSpace(binarySum) != strings.TrimSpace(checksum) {
		fmt.Printf("WARNING: Checksum for osdctl does not match the calculated value. Please retry installation. If issue persists, this tool can be downloaded manually at %s\n", osdctlBinaryAsset.GetBrowserDownloadURL())
		// We shouldn't link this binary to latest if the checksum isn't valid
		return nil
	}
	// Untar osdctl file
	err = unArchive(osdctlBinaryFilepath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive the osdctl asset file '%s': %w", filepath.Join(versionedDir, osdctlBinaryAsset.GetName()), err)
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'osdctl' binary at '%s': %w", latestDir, err)
	}
	err = os.Symlink(filepath.Join(versionedDir, "osdctl"), latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'osdctl' binary to '%s': %w", latestDir, err)
	}
	return nil
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "osdctl")
}

func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "osdctl")
}

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

func unArchive(source string, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open tarball '%s': %v", source, err)
	}
	defer func() {
		err = src.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close '%s': %v\n", src.Name(), err)
		}
	}()
	uncompressed, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to read the gzip file '%s': %v", source, err)
	}
	defer func() {
		err = uncompressed.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close gzip file '%s': %v", source, err)
		}
	}()
	arc := tar.NewReader(uncompressed)
	var f *tar.Header
	for {
		f, err = arc.Next()
		if err != io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read from archive '%s': %v", source, err)
		}
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(filepath.Join(destination, f.Name), 0755)
			if err != nil {
				return fmt.Errorf("failed to create a directory : %v", err)
			}
		} else {
			err = extractFile(destination, f, arc)
			if err != nil {
				return fmt.Errorf("failed to extract files: %v", err)
			}
		}
	}
	return nil
}

func (t *Tool) Configure() error {
	return nil
}

func extractFile(destination string, f *tar.Header, arc io.Reader) error {
	dst, err := os.Create(filepath.Join(destination, f.Name))
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer func() {
		err = dst.Close()
		if err != nil {
			fmt.Printf("warning: failed to close '%s': %v\n", dst.Name(), err)
		}
	}()

	err = dst.Chmod(os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to set permission on '%s': %v", dst.Name(), err)
	}
	_, err = dst.ReadFrom(arc)
	if err != nil {
		return fmt.Errorf("failed to read from archive  %v", err)
	}
	return nil
}
