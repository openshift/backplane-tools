package awscli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/openshift/backplane-tools/pkg/source/aws"
	"github.com/openshift/backplane-tools/pkg/source/github"
	"github.com/openshift/backplane-tools/pkg/utils"
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
	version, err := t.source.FetchLatestTag()
	if err != nil {
		return err
	}

	var (
		awsExecDir                 string
		awsOldInstallDir           string
		awsBinaryFilepath          string
		awsCompleterBinaryFilepath string
		url                        string
		fileExtension              string
	)

	toolDir := t.toolDir(rootDir)
	versionedDir := filepath.Join(toolDir, version)

	switch runtime.GOOS {
	case "linux":
		// Assign variables for Linux
		awsExecDir = "dist"
		fileExtension = ".zip"
		url = "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-" + version + fileExtension
	case "darwin":
		// Assign variables for macOS
		awsExecDir = "aws-cli.pkg/Payload/aws-cli"
		fileExtension = ".pkg"
		url = "https://awscli.amazonaws.com/AWSCLIV2" + fileExtension
	default:
		// Handle unsupported operating systems
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	err = os.RemoveAll(versionedDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Download the latest awscli
	err = os.MkdirAll(versionedDir, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	err = aws.DownloadAWSCLIRelease(url, fileExtension, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to download aws cli: %w", err)
	}

	// Unzip binary Bundle
	bundle := "aws-cli" + fileExtension
	awsArchiveFilepath := filepath.Join(versionedDir, bundle)
	awsNewInstallDir := filepath.Join(versionedDir, "aws-cli")

	if fileExtension == ".zip" {
		err = utils.Unzip(awsArchiveFilepath, versionedDir)
		if err != nil {
			return fmt.Errorf("failed to unarchive the aws-cli file '%s': %w", awsArchiveFilepath, err)
		}
		awsOldInstallDir = filepath.Join(versionedDir, "aws")
		// Rename unzipped directory
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
	}
	awsBinaryFilepath = filepath.Join(awsNewInstallDir, awsExecDir, "aws")
	awsCompleterBinaryFilepath = filepath.Join(awsNewInstallDir, awsExecDir, "aws_completer")

	// Link as latest
	latestFilePath := t.symlinkPath(latestDir)
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'aws' binary at '%s': %w", latestDir, err)
	}

	awsWrapperPath, err := t.createWrapper(versionedDir, awsBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to create aws cli squid proxy wrapper: %w", err)
	}

	err = os.Symlink(awsWrapperPath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'aws' binary to '%s': %w", latestDir, err)
	}

	// Link as latest also aws_completer
	latestCompleterFilePath := t.symlinkCompleterPath(latestDir)
	err = os.Remove(latestCompleterFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'aws_completer' binary at '%s': %w", latestDir, err)
	}

	err = os.Symlink(awsCompleterBinaryFilepath, latestCompleterFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'aws' binary to '%s': %w", latestDir, err)
	}

	return nil
}

func (t *Tool) Installed(rootDir string) (bool, error) {
	toolDir := t.toolDir(rootDir)
	return utils.FileExists(toolDir)
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Tool) toolDir(rootDir string) string {
	return filepath.Join(rootDir, "aws")
}

func (t *Tool) symlinkPath(latestDir string) string {
	return filepath.Join(latestDir, "aws")
}

func (t *Tool) symlinkCompleterPath(latestDir string) string {
	return filepath.Join(latestDir, "aws_completer")
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
	latestCompleterFilePath := t.symlinkCompleterPath(latestDir)
	err = os.Remove(latestCompleterFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'aws_completer' binary at '%s': %w", latestDir, err)
	}
	return nil
}

// Creates script that routes all aws traffic through squid proxy
func (t *Tool) createWrapper(versionedDir, awsPath string) (string, error) {
	var builder strings.Builder
	builder.WriteString(`#!/usr/bin/env bash
set \
  -o nounset \
  -o pipefail \
  -o errexit
export HTTPS_PROXY=squid.corp.redhat.com:3128
export HTTP_PROXY=squid.corp.redhat.com:3128

if ! command -v curl &> /dev/null
then
  # if curl isn't installed, print a warning
  echo "WARN: curl is not installed, cannot preflight VPN connection. If this command seems to hang you might need to connect to the VPN" 1>&2
else
  # Fail fast if not connected to the VPN instead of a very long timeout (exit code is curls proxy exit code)
  # This connect-timeout may need to be tuned depending on SRE geolocation and latency to the proxy
  if ! curl --connect-timeout 1 squid.corp.redhat.com > /dev/null 2>&1
  then
    echo "BPTools Error: Proxy Unavailable. Are you on the VPN?" 1>&2
    exit 5
  fi
fi
`)
	builder.WriteString(fmt.Sprintf("exec %s \"$@\"\n", awsPath))

	input := builder.String()
	awsWrapperPath := filepath.Join(versionedDir, "aws")

	err := os.WriteFile(awsWrapperPath, []byte(input), 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to create exec file: %w", err)
	}

	return awsWrapperPath, nil
}

func (t *Tool) Configure() error {
	return nil
}
