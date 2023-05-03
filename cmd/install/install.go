package install

import (
	"fmt"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tool"
	"github.com/openshift/backplane-tools/pkg/utils"
	"github.com/spf13/cobra"
)

// Cmd returns the Command used to invoke the installation logic
func Cmd() *cobra.Command {
	toolMap := tool.GetMap()
	installCmd := &cobra.Command{
		Use:       fmt.Sprintf("install [all|%s]", strings.Join(toolMap.Names(), "|")),
		Args:      cobra.OnlyValidArgs,
		ValidArgs: append(toolMap.Names(), "all"),
		Short:     "Install a new tool",
		Long:      "Installs one or more tools from the given list. It's valid to specify multiple tools: in this case, all tools provided will be installed. If no specific tools are provided, all are installed by default.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, toolMap)
		},
	}
	return installCmd
}

// run installs the tools specified by the provided positional args
func run(cmd *cobra.Command, args []string, toolMap tool.Map) error {
	if len(args) == 0 || utils.Contains(args, "all") {
		// If user doesn't specify, or explicitly passes 'all', give them all the things
		args = toolMap.Names()
	}

	fmt.Println("Installing the following tools:")
	installList := []tool.Tool{}
	for _, toolName := range args {
		fmt.Printf("- %s\n", toolName)
		installList = append(installList, toolMap[toolName])
	}

	err := tool.Install(installList)
	if err != nil {
		return fmt.Errorf("failed to install tools: %w", err)
	}
	return nil
}
