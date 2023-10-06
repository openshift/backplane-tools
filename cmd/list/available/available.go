package available

import (
	"fmt"

	"github.com/openshift/backplane-tools/pkg/tool"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	availableCmd := &cobra.Command{
		Use: "available",
		Args: cobra.NoArgs,
		Aliases: []string{"installable", "possible"},
		Short: "List available tools for install",
		Long: "List tools that are available to install with backplane-tools",
		RunE: func(_ *cobra.Command, _ []string) error {
			return List()
		},
	}
	return availableCmd
}

func List() error {
	fmt.Println("The following tools are available for install:")

	toolMap := tool.GetMap()
	for toolName := range toolMap {
		fmt.Printf("- %s\n", toolName)
	}
	return nil
}
