package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tools/awscli"
	"github.com/openshift/backplane-tools/pkg/tools/backplanecli"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/tools/butane"
	"github.com/openshift/backplane-tools/pkg/tools/gcloud"
	"github.com/openshift/backplane-tools/pkg/tools/oc"
	"github.com/openshift/backplane-tools/pkg/tools/ocm"
	"github.com/openshift/backplane-tools/pkg/tools/ocmaddons"
	"github.com/openshift/backplane-tools/pkg/tools/osdctl"
	"github.com/openshift/backplane-tools/pkg/tools/rosa"
	"github.com/openshift/backplane-tools/pkg/tools/self"
	"github.com/openshift/backplane-tools/pkg/tools/servicelogger"
	"github.com/openshift/backplane-tools/pkg/tools/yq"
	"github.com/openshift/backplane-tools/pkg/utils"
)

type Tool interface {
	// Name returns the name of the tool
	Name() string

	// ExecutableName returns the main binary name of the tool that will be installed in latest
	ExecutableName() string

	// Install fetches the latest tool from it's respective source, installs
	// it in a tool-unique directory under the provided rootDir, and symlinks
	// it to provided the latestDir
	Install() error

	// Configure currently unused
	Configure() error

	// Remove uninstalls the tool by deleting its tool-unique directory under
	// the provided rootDir and unlinking itself from the latestDir
	Remove() error

	// Installed validates whether the tool has already been installed under the
	// provided rootDir or not
	Installed() (bool, error)

	// InstalledVersion gets the version installed in latest folder
	InstalledVersion() (string, error)

	// LatestVersion gets the latest version available on repo
	LatestVersion() (string, error)
}

var toolMap map[string]Tool

func initMap() {
	toolMap = map[string]Tool{}

	// Self-management
	selfTool := self.New()
	toolMap[selfTool.Name()] = selfTool

	// 3rd party tools
	awsTool := awscli.New()
	toolMap[awsTool.Name()] = awsTool

	ocTool := oc.New()
	toolMap[ocTool.Name()] = ocTool

	ocmTool := ocm.New()
	toolMap[ocmTool.Name()] = ocmTool

	ocmaddonsTool := ocmaddons.New()
	toolMap[ocmaddonsTool.Name()] = ocmaddonsTool

	osdctlTool := osdctl.New()
	toolMap[osdctlTool.Name()] = osdctlTool

	backplanecliTool := backplanecli.New()
	toolMap[backplanecliTool.Name()] = backplanecliTool

	rosaTool := rosa.New()
	toolMap[rosaTool.Name()] = rosaTool

	yqTool := yq.New()
	toolMap[yqTool.Name()] = yqTool

	butaneTool := butane.New()
	toolMap[butaneTool.Name()] = butaneTool

	gcloudTool, err := gcloud.New()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Encountered error while initializing the 'gcloud' tool: %v\n", err)
		_, _ = fmt.Fprintln(os.Stderr, "Unable to install, upgrade, or remove 'gcloud' until the error is resolved")
	} else {
		toolMap[gcloudTool.Name()] = gcloudTool
	}

	serviceloggerTool := servicelogger.New()
	toolMap[serviceloggerTool.Name()] = serviceloggerTool
}

func GetMap() map[string]Tool {
	if toolMap == nil {
		initMap()
	}
	return toolMap
}

func Names() []string {
	return utils.Keys(GetMap())
}

// Remove removes the provided tools from the installation directory
func Remove(tools []Tool) error {
	for _, tool := range tools {
		fmt.Println()
		fmt.Printf("Removing %s\n", tool.Name())
		err := tool.Remove()
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
	err := createInstallDir()
	if err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Create the 'latest' directory which contains symlinks to the latest versions of each tool
	err = createLatestDir()
	if err != nil {
		return fmt.Errorf("failed to create latest directory: %w", err)
	}

	for _, tool := range tools {
		fmt.Println()
		fmt.Printf("Installing %s\n", tool.Name())
		err = tool.Install()
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
		fmt.Printf("WARNING: Couldn't determine current $PATH: it's recommended '%s' is added to your $PATH to utilize the tools provided by this application", base.LatestDir)
		return nil
	}
	userPaths := strings.Split(userPath, string(os.PathListSeparator))
	if !utils.Contains(userPaths, base.LatestDir) {
		fmt.Println()
		fmt.Printf("WARNING: Detected that '%s' is not present in $PATH: it's recommended '%s' is added to your $PATH to utilize the tools provided by this application\n", base.LatestDir, base.LatestDir)
	}
	return nil
}

func createInstallDir() error {
	return os.MkdirAll(base.InstallDir, os.FileMode(0o755))
}

func createLatestDir() error {
	return os.MkdirAll(base.LatestDir, os.FileMode(0o755))
}

func RemoveInstallDir() error {
	return os.RemoveAll(base.InstallDir)
}

// ListInstalled returns a slice containing all tools the current machine has installed
func ListInstalled() ([]Tool, error) {
	tools := GetMap()
	installedTools := make([]Tool, 0)

	for _, tool := range tools {
		installed, err := tool.Installed()
		if err != nil {
			return installedTools, err
		}
		if installed {
			installedTools = append(installedTools, tool)
		}
	}
	return installedTools, nil
}
