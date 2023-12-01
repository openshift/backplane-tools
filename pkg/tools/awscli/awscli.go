package awscli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/openshift/backplane-tools/pkg/sources/aws"
	"github.com/openshift/backplane-tools/pkg/sources/github"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool implements the interface to manage the 'aws-cli' binary
type Tool struct {
	base.Github
}

func New() *Tool {
	t := &Tool{
		Github: base.Github{
			Default:            base.Default{Name: "aws"},
			Source:             github.NewSource("aws", "aws-cli"),
			VersionInLatestTag: true,
		},
	}
	return t
}

func (t *Tool) Install() error {
	// Pull latest version from GH
	version, err := t.LatestVersion()
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

	toolDir := t.ToolDir()
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
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'aws' binary at '%s': %w", base.LatestDir, err)
	}

	awsWrapperPath, err := t.createWrapper(versionedDir, awsBinaryFilepath)
	if err != nil {
		return fmt.Errorf("failed to create aws cli squid proxy wrapper: %w", err)
	}

	err = os.Symlink(awsWrapperPath, latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'aws' binary to '%s': %w", base.LatestDir, err)
	}

	// Link as latest also aws_completer
	latestCompleterFilePath := t.symlinkCompleterPath()
	err = os.Remove(latestCompleterFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing 'aws_completer' binary at '%s': %w", base.LatestDir, err)
	}

	err = os.Symlink(awsCompleterBinaryFilepath, latestCompleterFilePath)
	if err != nil {
		return fmt.Errorf("failed to link new 'aws' binary to '%s': %w", base.LatestDir, err)
	}

	return nil
}

func (t *Tool) symlinkCompleterPath() string {
	return filepath.Join(base.LatestDir, "aws_completer")
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
