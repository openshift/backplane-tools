package aws_cli

import (
	"fmt"
	"github.com/openshift/backplane-tools/pkg/source/aws"
	"github.com/openshift/backplane-tools/pkg/source/github"
	"github.com/openshift/backplane-tools/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Tool implements the interface to manage the 'aws-cli' binary
type Tool struct {
	source *github.Source
}

func NewTool() *Tool {
	t := &Tool{
		source: github.NewSource("aws", "aws-cli"),
	}
	return t
}

func (t *Tool) Name() string {
	return "aws"
}

func (t *Tool) Install(rootDir, latestDir string) error {
	// Pull latest version from GH
	version, err := t.source.FetchTag()
	if err != nil {
		return err
	}

	var (
		awsExec           string
		awsOldInstallDir  string
		awsBinaryFilepath string
		url               string
		fileExtension     string
		operatingSys      string
	)

	toolDir := t.toolDir(rootDir)

	if runtime.GOOS == "linux" {
		// Assign variables for Linux
		awsExec = "dist/aws"
		operatingSys = "linux"
		fileExtension = ".zip"
		url = "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-" + version + fileExtension

	} else if runtime.GOOS == "darwin" {
		// Assign variables for macOS
		awsExec = "aws-cli.pkg/Payload/aws-cli/aws"
		operatingSys = "darwin"
		fileExtension = ".pkg"
		url = "https://awscli.amazonaws.com/AWSCLIV2" + fileExtension

	} else {
		// Handle unsupported operating systems
		return fmt.Errorf("Unsupported operating system:", runtime.GOOS)
	}

	err = os.RemoveAll(toolDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	//Download the latest awscli
	err = os.MkdirAll(toolDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", toolDir, err)
	}

	err = aws.DownloadAWSCLIRelease(url, fileExtension, toolDir)
	if err != nil {
		return fmt.Errorf("failed to download aws cli: %w", err)
	}

	//Unzip binary Bundle
	bundle := "aws-cli" + fileExtension
	awsArchiveFilepath := filepath.Join(toolDir, bundle)
	awsNewInstallDir := filepath.Join(toolDir, "aws-cli")

	if fileExtension == ".zip" {
		err = utils.Unzip(awsArchiveFilepath, toolDir)
		if err != nil {
			return fmt.Errorf("failed to unarchive the aws-cli file '%s': %w", awsArchiveFilepath, err)
		}
		awsOldInstallDir = filepath.Join(toolDir, "aws")
		awsBinaryFilepath = filepath.Join(awsNewInstallDir, awsExec)
		//Rename unzipped directory
		err = os.Rename(awsOldInstallDir, awsNewInstallDir)
		if err != nil {
			return fmt.Errorf("error renaming directory %w", err)

		}

	} else {
		cmd := exec.Command("pkgutil", "--expand-full", awsArchiveFilepath, awsNewInstallDir)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to extract the aws-cli file '%s': %w", awsArchiveFilepath, err)
		}
		awsBinaryFilepath = filepath.Join(awsNewInstallDir, awsExec)
	}

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'aws' binary at '%s': %w", latestDir, err)
	}

	err = t.createWrapper(latestFilePath, awsBinaryFilepath, toolDir, operatingSys)
	if err != nil {
		return fmt.Errorf("failed to create aws cli squid proxy wrapper: %w", err)
	}

	return nil
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "aws")
}

func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "aws")
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

func (t *Tool) createWrapper(latestDir, awsPath, toolDir, operatingSys string) error {
	var builder strings.Builder
	builder.WriteString(`#!/usr/bin/env bash
set \
  -o nounset \
  -o pipefail \
  -o errexit
export HTTPS_PROXY=squid.corp.redhat.com:3128
export HTTP_PROXY=squid.corp.redhat.com:3128
`)
	builder.WriteString(fmt.Sprintf("exec %s \"$@\"\n", awsPath))

	input := builder.String()

	err := os.WriteFile(latestDir, []byte(input), 0755)
	if err != nil {
		return fmt.Errorf("failed to create exec file: %v", err)
	}

	return nil
}

func (t *Tool) Configure() error {
	return nil
}
