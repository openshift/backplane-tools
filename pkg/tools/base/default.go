package base

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift/backplane-tools/pkg/utils"
)

type Default struct {
	Tool
	Name             string
	executableName   string // if empty, name will be used
	installedVersion string
	latestVersion    string
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Default) ToolDir() string {
	return filepath.Join(InstallDir, t.Name)
}

// symlinkPath returns the path to the symlink created by this tool, given the latest directory
func (t *Default) SymlinkPath() string {
	return filepath.Join(LatestDir, t.GetExecutableName())
}

// Name returns the name of the tool
func (t *Default) GetName() string {
	return t.Name
}

// SetExecutableName to initialize private field executableName and not use Name
func (t *Default) SetExecutableName(executableName string) {
	t.executableName = executableName
}

// ExecutableName returns the main executable name of the tool that will be installed in latest
// if not defined it will be the name
func (t *Default) GetExecutableName() string {
	if t.executableName == "" {
		return t.Name
	}
	return t.executableName
}

// Confiure is currently unused
func (t *Default) Configure() error {
	return nil
}

// Remove uninstalls the tool by deleting it's tool-unique directory under
// the provided rootDir and unlinking itself from the latestDir
func (t *Default) Remove() error {
	// Remove all binaries owned by this tool
	toolDir := t.ToolDir()
	err := os.RemoveAll(toolDir)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", toolDir, err)
	}

	// Remove all symlinks owned by this tool
	latestFilePath := t.SymlinkPath()
	err = os.Remove(latestFilePath)
	if err != nil {
		return fmt.Errorf("failed to remove symlinked file %s: %w", latestFilePath, err)
	}
	return nil
}

// Installed validates whether the tool has already been installed under the
// provided rootDir or not
func (t *Default) Installed() (bool, error) {
	toolDir := t.ToolDir()
	return utils.FileExists(toolDir)
}

func (t *Default) InstalledVersion() (string, error) {
	if t.installedVersion == "" {
		latestFilePath := t.SymlinkPath()
		latestFileTarget, err := filepath.EvalSymlinks(latestFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve symlinked file %s: %w", latestFilePath, err)
		}
		rootDirTarget, err := filepath.EvalSymlinks(InstallDir)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve rootDir folder %s: %w", rootDirTarget, err)
		}
		relLatestFileTarget, err := filepath.Rel(rootDirTarget, latestFileTarget)
		if err != nil {
			return "", fmt.Errorf("failed to convert latestFilePath %s to relative: %w", latestFileTarget, err)
		}
		t.installedVersion = strings.SplitN(relLatestFileTarget, string(os.PathSeparator), 3)[1]
	}
	return t.installedVersion, nil
}
