package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tool/ocm"
	"github.com/openshift/backplane-tools/pkg/tool/osdctl"
	"github.com/openshift/backplane-tools/pkg/utils"
)

// Tool defines the implementation to install, remove, configure, etc a single application
type Tool interface {
	// Name returns the name this tool should be referred to by users
	Name() string

	// Install contains the application-specific logic required to install a new tool
	Install(rootDir, latestDir string) error

	// Remove contains the application-specific logic required to remove a new tool
	Remove(rootDir, latestDir string) error
}

// Map is a convenience type used to simplify Tool lookup tables
type Map map[string]Tool

// Names returns the names of the tools contained within the Map
func (m Map) Names() []string {
	return utils.Keys(m)
}

// toolMap represents the global internal lookup table. It is used within this package to
// track the various tools managed by this application
var toolMap Map

// newMap creates the internal toolMap object
func newMap() Map {
	toolMap = Map{}

	ocmTool := ocm.NewTool()
	toolMap[ocmTool.Name()] = ocmTool
	osdctlTool := osdctl.NewTool()
	toolMap[osdctlTool.Name()] = osdctlTool
	return toolMap
}

// GetMap returns the global Tool lookup table; initializing it first, if necessary
func GetMap() Map {
	if toolMap == nil {
		return newMap()
	}
	return toolMap
}

// Remove removes the provided tools from the install directory
func Remove(tools []Tool) error {
	installDir, err := InstallDir()
	if err != nil {
		return fmt.Errorf("failed to get installation directory: %w", err)
	}

	latestDir, err := LatestDir()
	if err != nil {
		return fmt.Errorf("failed to get latest release directory: %w", err)
	}

	for _, tool := range tools {
		fmt.Println()
		fmt.Printf("Removing %s\n", tool.Name())
		err = tool.Remove(installDir, latestDir)
		if err != nil {
			fmt.Printf("Encountered error while removing %s: %v\n", tool.Name(), err)
			fmt.Println("Skipping...")
		} else {
			fmt.Printf("Successfully removed %s\n", tool.Name())
		}
	}
	return nil
}

// Install builds the basic directory structure and installs the provided tools
func Install(tools []Tool) error {
	// Create the root directory for all tools to install into
	installDir, err := CreateInstallDir()
	if err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Create the 'latest' directory which contains symlinks to the latest versions of each tool
	latestDir, err := createLatestDir()
	if err != nil {
		return fmt.Errorf("failed to create latest directory: %w", err)
	}

	for _, tool := range tools {
		fmt.Println()
		fmt.Printf("Installing %s\n", tool.Name())
		err = tool.Install(installDir, latestDir)
		if err != nil {
			fmt.Printf("Encountered error while installing %s: %v\n", tool.Name(), err)
			fmt.Println("Skipping...")
		} else {
			fmt.Printf("Successfully installed %s\n", tool.Name())
		}
	}

	// Check $PATH for the latest binaries
	userPath, found := os.LookupEnv("PATH")
	if !found {
		fmt.Println()
		fmt.Printf("WARNING: Couldn't determine current $PATH: it's recommended '%s' is added to your $PATH to utilize the tools provided by this application", latestDir)
		return nil
	}
	userPaths := strings.Split(userPath, string(os.PathListSeparator))
	if !utils.Contains(userPaths, latestDir) {
		fmt.Println()
		fmt.Printf("WARNING: Detected that '%s' is not present in $PATH: it's recommended '%s' is added to your $PATH to utilize the tools provided by this application\n", latestDir, latestDir)
	}
	return nil
}

// InstallDir returns the installation directory used by this application
func InstallDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve $HOME dir: %w", err)
	}
	return filepath.Join(homeDir, ".local", "bin", "backplane"), nil
}

// CreateInstallDir creates the installation directory used by this application
func CreateInstallDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", fmt.Errorf("could not determine install path: %w", err)
	}
	err = os.MkdirAll(installDir, os.FileMode(0755))
	return installDir, err
}

// RemoveInstallDir deletes the installation directory used by this application
func RemoveInstallDir() error {
	installDir, err := InstallDir()
	if err != nil {
		return fmt.Errorf("could not determine install path: %w", err)
	}
	return os.RemoveAll(installDir)
}

// LatestDir returns the directory containing the latest versions of tools managed by this application
func LatestDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", fmt.Errorf("could not determine install path: %w", err)
	}
	return filepath.Join(installDir, "latest"), nil
}

// createLatestDir creates the directory containing the latest versions of tools managed by this application
func createLatestDir() (string, error) {
	latestDir, err := LatestDir()
	if err != nil {
		return "", fmt.Errorf("could not determine latest release path: %w", err)
	}
	err = os.MkdirAll(latestDir, os.FileMode(0755))
	return latestDir, err
}
