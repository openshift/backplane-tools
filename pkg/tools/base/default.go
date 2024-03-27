package base

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift/backplane-tools/pkg/utils"
)

var InstallDir = func() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to retrieve $HOME dir: %w", err))
	}
	return filepath.Join(homeDir, ".local", "bin", "backplane")
}()

var LatestDir = func() string {
	return filepath.Join(InstallDir, "latest")
}()

type Default struct {
	// Name defines the 'formal' name this tool is referred to within this program
	name string

	// executableName defines the invokable used to run the tool via CLI after it has
	// been installed
	executableName string

	// installedVersion is the currently installed version of the tool
	installedVersion string

	// latestVersion is the latest version of the tool available for install
	latestVersion string

	// oneShot indicates whether the tool should only have it's install logic run once
	oneShot bool

	// oneShotHelp indicates how the user should manage the tool once it has been bootstrapped by backplane-tools
	oneShotHelp string
}

// NewDefault creates a Default tool with the provided name
func NewDefault(name string) Default {
	d := Default{
		name:           name,
		executableName: name,
	}
	return d
}

// NewDefaultWithExecutable creates a Default tool with the provided name and executableName
func NewDefaultWithExecutable(name, executable string) Default {
	d := Default{
		name:           name,
		executableName: executable,
	}
	return d
}

func NewDefaultOneShot(name, help string) Default {
	d := Default{
		name: name,
		oneShot: true,
		oneShotHelp: help,
	}
	return d
}

// toolDir returns this tool's specific directory given the root directory all tools are installed in
func (t *Default) ToolDir() string {
	return filepath.Join(InstallDir, t.name)
}

// symlinkPath returns the path to the symlink created by this tool, given the latest directory
func (t *Default) SymlinkPath() string {
	return filepath.Join(LatestDir, t.executableName)
}

// Name returns the name of the tool
func (t *Default) Name() string {
	return t.name
}

// ExecutableName returns the main executable name of the tool that will be installed in latest
// if not defined it will be the name
func (t *Default) ExecutableName() string {
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

// InstalledVersion returns the currently installed version of the tool
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

// OneShot indicates whether the tool should only be installed once
func (t *Default) OneShot() bool {
	return t.oneShot
}

// OneShotHelp indicates how the user should manage the tool once it has been bootstrapped by backplane-tools
func (t *Default) OneShotHelp() string {
	return t.oneShotHelp
}
