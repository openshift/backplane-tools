package upgrade

import (
	"fmt"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tools"
	"github.com/openshift/backplane-tools/pkg/tools/base"
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
	var listTools []base.Tool
	if len(args) == 0 || utils.Contains(args, "all") {
		// If user explicitly passes 'all' or doesn't specify which tools to install,
		// upgrade everything that's been installed locally
		var err error
		listTools, err = tools.ListInstalled()
		if err != nil {
			return err
		}
	} else {
		// otherwise build the list verifying tool exist
		toolMap := tools.GetMap()

		listTools = []base.Tool{}
		for _, toolName := range args {
			t, found := toolMap[toolName]
			if !found {
				return fmt.Errorf("failed to locate '%s' in list of supported tools", toolName)
			}
			listTools = append(listTools, t)
		}
	}

	fmt.Println("Upgrading the following tools: ")
	upgradeList := []base.Tool{}
	for _, t := range listTools {
		latestVersion, err := t.LatestVersion()
		if err != nil {
			return fmt.Errorf("failed to determine version for '%s': %w", t.GetName(), err)
		}
		installedVersion, err := t.InstalledVersion()
		if err != nil {
			return fmt.Errorf("failed to determine version for '%s': %w", t.GetName(), err)
		}
		if installedVersion == latestVersion {
			fmt.Printf("- %s is already installed with latest version %s and will not be upgraded\n", t.GetName(), latestVersion)
		} else {
			upgradeList = append(upgradeList, t)
			fmt.Printf("- %s %s -> %s\n", t.GetName(), installedVersion, latestVersion)
		}
	}

	err := tools.Install(upgradeList)
	if err != nil {
		return fmt.Errorf("failed to upgrade tools: %w", err)
	}
	return nil
}
