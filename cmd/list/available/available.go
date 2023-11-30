package available

import (
	"fmt"

	"github.com/openshift/backplane-tools/pkg/tools"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	availableCmd := &cobra.Command{
		Use:     "available",
		Args:    cobra.NoArgs,
		Aliases: []string{"installable", "possible"},
		Short:   "List available tools for install",
		Long:    "List tools that are available to install with backplane-tools",
		RunE: func(_ *cobra.Command, _ []string) error {
			return List()
		},
	}
	return availableCmd
}

func List() error {
	fmt.Println("The following tools are available for install:")

	toolMap := tools.GetMap()
	for _, t := range toolMap {
		version, err := t.LatestVersion()
		if err != nil {
			return fmt.Errorf("failed to determine version for '%s': %w", t.GetName(), err)
		}
		fmt.Printf("- %s %s\n", t.GetName(), version)
	}
	return nil
}
