package install

import (
	"fmt"
	"strings"

	"github.com/openshift/backplane-tools/pkg/tools"
	"github.com/openshift/backplane-tools/pkg/tools/base"
	"github.com/openshift/backplane-tools/pkg/utils"
	"github.com/spf13/cobra"
)

// Cmd returns the Command used to invoke the installation logic
func Cmd() *cobra.Command {
	toolNames := tools.Names()
	installCmd := &cobra.Command{
		Use:       fmt.Sprintf("install [all|%s]", strings.Join(toolNames, "|")),
		Args:      cobra.OnlyValidArgs,
		ValidArgs: append(toolNames, "all"),
		Short:     "Install a new tool",
		Long:      "Installs one or more tools from the given list. It's valid to specify multiple tools: in this case, all tools provided will be installed. If no specific tools are provided, all are installed by default.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Install(args)
		},
	}
	return installCmd
}

// run installs the tools specified by the provided positional args
func Install(args []string) error {
	fmt.Println("Installing the following tools:")
	toolMap := tools.GetMap()
	installList := []base.Tool{}
	if len(args) == 0 || utils.Contains(args, "all") {
		// If user doesn't specify, or explicitly passes 'all', give them all the things
		for _, tool := range toolMap {
			installList = append(installList, tool)
		}
	} else {
		for _, toolName := range args {
			installList = append(installList, toolMap[toolName])
		}
	}
	for _, tool := range installList {
		latestversion, err := tool.LatestVersion()
		if err != nil {
			return err
		}
		fmt.Printf("- %s %s\n", tool.GetName(), latestversion)
	}

	err := tools.Install(installList)
	if err != nil {
		return fmt.Errorf("failed to install tools: %w", err)
	}
	return nil
}
