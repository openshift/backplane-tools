package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tools/awscli"
	"github.com/openshift/backplane-tools/pkg/tools/backplanecli"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/tools/oc"
	"github.com/openshift/backplane-tools/pkg/tools/ocm"
	"github.com/openshift/backplane-tools/pkg/tools/osdctl"
	"github.com/openshift/backplane-tools/pkg/tools/rosa"
	"github.com/openshift/backplane-tools/pkg/tools/self"
	"github.com/openshift/backplane-tools/pkg/tools/yq"
	"github.com/openshift/backplane-tools/pkg/utils"
)

var toolMap map[string]base.Tool

func initMap() {
	toolMap = map[string]base.Tool{}

	// Self-management
	self := self.New()
	toolMap[self.GetName()] = self

	// 3rd party tools
	aws := awscli.New()
	toolMap[aws.GetName()] = aws

	oc := oc.New()
	toolMap[oc.GetName()] = oc

	ocm := ocm.New()
	toolMap[ocm.GetName()] = ocm

	osdctl := osdctl.New()
	toolMap[osdctl.GetName()] = osdctl

	backplanecli := backplanecli.New()
	toolMap[backplanecli.GetName()] = backplanecli

	rosa := rosa.New()
	toolMap[rosa.GetName()] = rosa

	yq := yq.New()
	toolMap[yq.GetName()] = yq
}

func GetMap() map[string]base.Tool {
	if toolMap == nil {
		initMap()
	}
	return toolMap
}

func Names() []string {
	return utils.Keys(GetMap())
}

// Remove removes the provided tools from the install directory
func Remove(tools []base.Tool) error {
	for _, tool := range tools {
		fmt.Println()
		fmt.Printf("Removing %s\n", tool.GetName())
		err := tool.Remove()
		if err != nil {
			fmt.Printf("Encountered error while removing %s: %v\n", tool.GetName(), err)
			fmt.Println("Skipping...")
		} else {
			fmt.Printf("Successfully removed %s\n", tool.GetName())
		}
	}
	return nil
}

// Install creates the directories necessary to install the provided tools and
func Install(tools []base.Tool) error {
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
		fmt.Printf("Installing %s\n", tool.GetName())
		err = tool.Install()
		if err != nil {
			fmt.Printf("Encountered error while installing %s: %v\n", tool.GetName(), err)
			fmt.Println("Skipping...")
		} else {
			fmt.Printf("Successfully installed %s\n", tool.GetName())
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
func ListInstalled() ([]base.Tool, error) {
	tools := GetMap()
	installedTools := []base.Tool{}

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
