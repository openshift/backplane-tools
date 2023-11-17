package base

import (
	"fmt"
	"os"
	"path/filepath"
)

func installDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to retrieve $HOME dir: %w", err))
	}
	return filepath.Join(homeDir, ".local", "bin", "backplane")
}

var InstallDir = installDir()

func latestDir() string {
	installDir := InstallDir
	return filepath.Join(installDir, "latest")
}

var LatestDir = latestDir()

type Tool interface {
	// Name returns the name of the tool
	GetName() string

	// ExecutableName returns the main binary name of the tool that will be installed in latest
	GetExecutableName() string

	// Install fetches the latest tool from it's respective source, installs
	// it in a tool-unique directory under the provided rootDir, and symlinks
	// it to provided the latestDir
	Install() error

	// Confiure is currently unused
	Configure() error

	// Remove uninstalls the tool by deleting it's tool-unique directory under
	// the provided rootDir and unlinking itself from the latestDir
	Remove() error

	// Installed validates whether the tool has already been installed under the
	// provided rootDir or not
	Installed() (bool, error)

	// Get the version installed in latest folder
	InstalledVersion() (string, error)

	// Get the latest version available on repo
	LatestVersion() (string, error)
}
