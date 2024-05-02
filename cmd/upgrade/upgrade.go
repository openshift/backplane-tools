package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tools"
	"github.com/openshift/backplane-tools/pkg/tools/self"
	"github.com/openshift/backplane-tools/pkg/utils"
	"github.com/spf13/cobra"
)

// Cmd returns the Command used to invoke the upgrade logic
func Cmd() *cobra.Command {
	toolNames := tools.Names()
	upgradeCmd := &cobra.Command{
		Use:       fmt.Sprintf("upgrade [all|%s]", strings.Join(toolNames, "|")),
		Args:      cobra.OnlyValidArgs,
		ValidArgs: append(toolNames, "all"),
		Short:     "Upgrade an existing tool",
		Long:      "Upgrades one or more tools from the provided list. It's valid to specify multiple tools: in this case, all tools provided will be upgraded. If no specific tools are provided, all are (installed and) upgraded by default.",
		RunE: func(_ *cobra.Command, args []string) error {
			return Upgrade(args)
		},
	}
	return upgradeCmd
}

// Upgrade upgrades the provided tools to their latest versions
func Upgrade(args []string) error {
	if len(args) == 0 || utils.Contains(args, "all") {
		// If user explicitly passes 'all' or doesn't specify which tools to install,
		// upgrade backplane-tools itself, then everything that's been installed locally
		err := upgradeAll()
		if err != nil {
			return err
		}
		return nil
	}
	// otherwise build the list verifying tool exist
	toolMap := tools.GetMap()

	listTools := []tools.Tool{}
	for _, toolName := range args {
		t, found := toolMap[toolName]
		if !found {
			return fmt.Errorf("failed to locate '%s' in list of supported tools", toolName)
		}
		listTools = append(listTools, t)
	}
	err := upgradeTools(listTools)
	if err != nil {
		return err
	}

	return nil
}

func upgradeAll() error {
	upgraded, err := upgradeSelfIfNecessary()
	if err != nil {
		return err
	}
	if upgraded {
		return rerunUpgradeAll()
	}
	listTools, err := tools.ListInstalled()
	if err != nil {
		return err
	}
	err = upgradeTools(listTools)
	if err != nil {
		return err
	}
	return nil
}

func upgradeTools(listTools []tools.Tool) error {
	fmt.Println("Upgrading the following tools: ")
	upgradeList := []tools.Tool{}
	for _, t := range listTools {
		latestVersion, err := t.LatestVersion()
		if err != nil {
			return fmt.Errorf("failed to determine version for '%s': %w", t.Name(), err)
		}
		installedVersion, err := t.InstalledVersion()
		if err != nil {
			return fmt.Errorf("failed to determine version for '%s': %w", t.Name(), err)
		}
		if installedVersion == latestVersion {
			fmt.Printf("- %s is already installed with latest version %s and will not be upgraded\n", t.Name(), latestVersion)
		} else {
			upgradeList = append(upgradeList, t)
			fmt.Printf("- %s %s -> %s\n", t.Name(), installedVersion, latestVersion)
		}
	}

	err := tools.Install(upgradeList)
	if err != nil {
		return fmt.Errorf("failed to upgrade tools: %w", err)
	}
	return nil
}

func upgradeSelfIfNecessary() (bool, error) {
	selfTool := self.New()
	latestVersion, err := selfTool.LatestVersion()
	if err != nil {
		return false, fmt.Errorf("failed to determine version for '%s': %w", selfTool.Name(), err)
	}
	installedVersion, err := selfTool.InstalledVersion()
	if err != nil {
		return false, fmt.Errorf("failed to determine version for '%s': %w", selfTool.Name(), err)
	}

	if installedVersion == latestVersion {
		return false, nil
	}
	fmt.Printf("upgrading %s %s -> %s\n", selfTool.Name(), installedVersion, latestVersion)
	err = tools.Install([]tools.Tool{selfTool})

	if err != nil {
		return false, fmt.Errorf("failed to upgrade backplane-tools: %w", err)
	}
	return true, nil
}

func rerunUpgradeAll() error {
	fmt.Printf("re-running '%s' to upgrade all tools...\n", strings.Join(os.Args, " "))
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run '%s': %w", strings.Join(os.Args, " "), err)
	}
	return nil
}
