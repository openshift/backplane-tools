package remove

import (
	"fmt"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tool"
	"github.com/openshift/backplane-tools/pkg/utils"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	toolMap := tool.GetMap()
	removeCmd := &cobra.Command{
		Use:       fmt.Sprintf("remove [all|%s]", strings.Join(toolMap.Names(), "|")),
		Args:      cobra.OnlyValidArgs,
		ValidArgs: append(toolMap.Names(), "all"),
		Short:     "Remove a tool",
		Long:      "Removes one or more tools from the given list. It's valid to specify multiple tools: in this case, all tools provided will be removed. If 'all' is explicitly passed, then the entire tool directory will be removed, providing a clean slate for reinstall. If no specific tools are provided, no action is taken",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, toolMap)
		},
	}
	return removeCmd
}

// run removes the tool(s) specified by the provided positional args
func run(cmd *cobra.Command, args []string, toolMap tool.Map) error {
	if len(args) == 0 {
		fmt.Println("No tools specified to be removed. In order to remove all tools, explicitly specify 'all'")
		return nil
	}
	if utils.Contains(args, "all") {
		return tool.RemoveInstallDir()
	}

	fmt.Println("Removing the following tools:")
	removeList := []tool.Tool{}
	for _, toolName := range args {
		fmt.Printf("- %s\n", toolName)
		removeList = append(removeList, toolMap[toolName])
	}

	err := tool.Remove(removeList)
	if err != nil {
		return fmt.Errorf("failed to remove one or more tools: %w", err)
	}
	return nil
}
