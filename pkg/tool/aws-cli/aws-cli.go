package aws_cli

import (
	"fmt"
	"github.com/openshift/backplane-tools/pkg/source/aws"
	"github.com/openshift/backplane-tools/pkg/source/github"
	"github.com/openshift/backplane-tools/pkg/utils"
	"os"
	"path/filepath"
)

// Tool implements the interface to manage the 'aws-cli' binary
type Tool struct {
	source *github.Source
}

func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("aws-cli", "aws-cli-cli"),
	}
	return t
}

func (t *Tool) Name() string {
	return "aws-cli"
}

func (t *Tool) Install(rootDir, latestDir string) error {
	// Pull latest version from GH
	version, err := t.source.FetchTag()
	if err != nil {
		return err
	}

	awsExec := "/dist/aws-cli"
	toolDir := t.toolDir(rootDir)
	versionedDir := filepath.Join(toolDir, version)
	awsCliPath := filepath.Join(versionedDir, "aws-cli-cli"+awsExec)

	// Check if aws-cli is already installed
	_, err = os.Stat(awsCliPath)
	if err == nil {
		fmt.Printf("'%s' is the most recent aws-cli version.\n", awsCliPath)
		return nil
	}

	//Download the latest awscli
	err = os.MkdirAll(versionedDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = aws.DownloadAWSCLIRelease(version, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download aws-cli-cli: %w", err)
	}

	//Unzip binary Bundle
	bundle := "aws-cli-cli.zip"
	awsArchiveFilepath := filepath.Join(versionedDir, bundle)
	err = utils.Unzip(awsArchiveFilepath, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to unarchive the aws-cli zip file '%s': %w", awsArchiveFilepath, err)
	}

	awsOldInstallDir := filepath.Join(versionedDir, "aws-cli")
	awsNewInstallDir := filepath.Join(versionedDir, "aws-cli-cli")
	err = os.Rename(awsOldInstallDir, awsNewInstallDir)
	if err != nil {
		return fmt.Errorf("error renaming directory %w", err)
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'aws-cli' binary at '%s': %w", latestDir, err)
	}

	awsBinaryFilepath := filepath.Join(awsNewInstallDir, awsExec)

	err = t.createWrapper(latestFilePath, awsBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to link new 'aws-cli' binary to '%s': %w", latestDir, err)
	}

	return nil
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "aws-cli")
}

func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "aws-cli")
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

func (t *Tool) createWrapper(latestDir, awsPath string) error {
	input := fmt.Sprintf(`#!/usr/bin/env bash
set \
  -o nounset \
  -o pipefail \
  -o errexit

export HTTPS_PROXY=squid.corp.redhat.com:3128
export HTTP_PROXY=squid.corp.redhat.com:3128

exec %s "$@"`, awsPath)

	err := os.WriteFile(latestDir, []byte(input), 0755)
	if err != nil {
		return fmt.Errorf("failed to create exec file: %v", err)
	}

	return nil
}

func (t *Tool) Configure() error {
	return nil
}
