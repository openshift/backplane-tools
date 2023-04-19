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
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, toolMap)
		},
	}
	return upgradeCmd
}

func run(cmd *cobra.Command, args []string, toolMap tool.Map) error {
	if len(args) == 0 || utils.Contains(args, "all") {
		// If user doesn't specify, or explicitly passes 'all', upgrade all the things
		args = toolMap.Names()
	}

	fmt.Println("Upgrading the following tools: ")
	upgradeList := []tool.Tool{}
	for _, toolName := range args {
		fmt.Printf("- %s\n", toolName)
		upgradeList = append(upgradeList, toolMap[toolName])
	}

	err := tool.Install(upgradeList)
	if err != nil {
		return fmt.Errorf("failed to upgrade tools: %w", err)
	}
	return nil
}
