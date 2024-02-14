package gcloud

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gstorage "cloud.google.com/go/storage"

	"github.com/openshift/backplane-tools/pkg/sources/cloud.google.com/storage"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/utils"
)

const (
	// toolBucket refers to the name of the bucket in cloud.google.com that contains the tool's assets
	// https://console.cloud.google.com/storage/browser/cloud-sdk-release;tab=objects?prefix=&forceOnObjectsSortingFiltering=false for reference
	toolBucket = "cloud-sdk-release"
	// listPrefix is a filter used to identify the tool's objects within the bucket
	listPrefix = "google-cloud-cli"
)

// Tool manages the installation, upgrade and removal of the 'gcloud' tool
type Tool struct {
	// Default defines the default Tool implementation
	base.Default
	// Source defines the source of the tool in cloud.google.com/storage
	Source *storage.Source
}

// New initializes a new 'gcloud' tool
func New() (*Tool, error) {
	src, err := storage.NewSource(toolBucket)
	if err != nil {
		return &Tool{}, fmt.Errorf("error while initializing gcloud bucket: %w", err)
	}
	t := &Tool{
		Default: base.NewDefault("gcloud"),
		Source:  src,
	}
	return t, nil
}

// Install installs a new gcloud tool on the local system
func (t *Tool) Install() error {
	// fetch info regarding the latest version of the tool
	latestArchive, err := t.findLatestObjectForSystem()
	if err != nil {
		return fmt.Errorf("failed to locate latest archive matching system spec: %w", err)
	}
	versionName, _ := t.getVersionNameFromArchive(latestArchive)

	// Create the tool- and version-specific directories for this install
	toolDir := t.ToolDir()
	versionedDir := filepath.Join(toolDir, versionName)
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	// Download the tool and un-tar it
	err = t.Source.DownloadObject(latestArchive, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download object '%s': %w", latestArchive.Name, err)
	}

	archiveFilePath := filepath.Join(versionedDir, latestArchive.Name)
	err = utils.Unarchive(archiveFilePath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive '%s': %w", archiveFilePath, err)
	}

	// Link as latest
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing symlink '%s': %w", latestFilePath, err)
	}

	executableFilePath := filepath.Join(versionedDir, "google-cloud-sdk", "bin", "gcloud")
	err = os.Symlink(executableFilePath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new executable '%s' to '%s': %w", executableFilePath, latestFilePath, err)
	}
	return nil
}

// LatestVersion determines the latest version of the tool available for install
func (t *Tool) LatestVersion() (string, error) {
	latestArchive, err := t.findLatestObjectForSystem()
	if err != nil {
		return "", fmt.Errorf("failed to locate latest archive matching system spec: %w", err)
	}
	version, _ := t.getVersionNameFromArchive(latestArchive)
	return version, nil
}

// findLatestObjectForSystem locates the most recent version of the gcloud tool based on the system's spec (OS+architecture)
func (t *Tool) findLatestObjectForSystem() (*gstorage.ObjectAttrs, error) {
	objs, err := t.Source.ListObjects(listPrefix)
	if err != nil {
		return &gstorage.ObjectAttrs{}, fmt.Errorf("failed to list objects in bucket: %w", err)
	}

	matches := t.Source.FindObjectsForArchAndOS(objs)
	if len(matches) == 0 {
		return &gstorage.ObjectAttrs{}, fmt.Errorf("unexpected number of assets found matching system spec: expected at least 1, got %d", len(matches))
	}

	return t.Source.FindLatest(matches), nil
}

// getVersionNameFromArchive is a helper function to convert a bucket object's name from <versioned-name>.tar.gz format to <versioned-name>
func (t *Tool) getVersionNameFromArchive(archive *gstorage.ObjectAttrs) (version string, found bool) {
	return strings.CutSuffix(archive.Name, ".tar.gz")
}
