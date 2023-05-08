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

type Tool interface {
	Name() string

	Install(rootDir, latestDir string) error

	Configure() error

	Remove(rootDir, latestDir string) error
}

type Map map[string]Tool

func (m Map) Names() []string {
	return utils.Keys(m)
}

var toolMap Map

func newMap() Map {
	toolMap = Map{}

	ocmTool := ocm.NewTool()
	toolMap[ocmTool.Name()] = ocmTool
	osdctlTool := osdctl.NewTool()
	toolMap[osdctlTool.Name()] = osdctlTool
	return toolMap
}

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

// Install creates the directories necessary to install the provided tools and
func Install(tools []Tool) error {
	// Create the root directory for all tools to install into
	installDir, err := createInstallDir()
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

func InstallDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve $HOME dir: %w", err)
	}
	return filepath.Join(homeDir, ".local", "bin", "backplane"), nil
}

func createInstallDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", fmt.Errorf("could not determine install path: %w", err)
	}
	err = os.MkdirAll(installDir, os.FileMode(0755))
	return installDir, err
}

func LatestDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", fmt.Errorf("could not determine install path: %w", err)
	}
	return filepath.Join(installDir, "latest"), nil
}

func createLatestDir() (string, error) {
	latestDir, err := LatestDir()
	if err != nil {
		return "", fmt.Errorf("could not determine latest release path: %w", err)
	}
	err = os.MkdirAll(latestDir, os.FileMode(0755))
	return latestDir, err
}

func RemoveInstallDir() error {
	installDir, err := InstallDir()
	if err != nil {
		return fmt.Errorf("could not determine install path: %w", err)
	}
	return os.RemoveAll(installDir)
}
