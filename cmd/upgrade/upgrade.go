package upgrade

import (
	"fmt"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tool"
	"github.com/openshift/backplane-tools/pkg/utils"
	"github.com/spf13/cobra"
)

// Cmd returns the Command used to invoke the upgrade logic
func Cmd() *cobra.Command {
	toolMap := tool.GetMap()

	upgradeCmd := &cobra.Command{
		Use:       fmt.Sprintf("upgrade [all|%s]", strings.Join(toolMap.Names(), "|")),
		Args:      cobra.OnlyValidArgs,
		ValidArgs: append(toolMap.Names(), "all"),
		Short:     "Upgrade an existing tool",
		Long:      "Upgrades one or more tools from the provided list. It's valid to specify multiple tools: in this case, all tools provided will be upgraded. If no specific tools are provided, all are (installed and) upgraded by default.",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || utils.Contains(args, "all") {
				// If user explicitly passes 'all' or doesn't specify which tools to install,
				// upgrade everything that's been installed locally
				installedTools, err := tool.ListInstalled()
				if err != nil {
					return err
				}
				args = []string{}
				for _, installedTool := range installedTools {
					args = append(args, installedTool.Name())
				}
			}
			return Upgrade(args)
		},
	}
	return upgradeCmd
}


// Upgrade upgrades the provided tools to their latest versions
func Upgrade(tools []string) error {
	toolMap := tool.GetMap()

	upgradeList := []tool.Tool{}
	for _, toolName := range tools {
		t, found := toolMap[toolName]
		if !found {
			return fmt.Errorf("failed to locate '%s' in list of supported tools", toolName)
		}
		upgradeList = append(upgradeList, t)
	}

	fmt.Println("Upgrading the following tools: ")
	for _, t := range upgradeList {
		fmt.Printf("- %s\n", t.Name())
	}

	err := tool.Install(upgradeList)
	if err != nil {
		return fmt.Errorf("failed to upgrade tools: %w", err)
	}
	return nil
}
