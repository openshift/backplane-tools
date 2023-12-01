package installed

import (
	"fmt"

	"github.com/openshift/backplane-tools/pkg/tools"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	installedCmd := &cobra.Command{
		Use:   "installed",
		Args:  cobra.NoArgs,
		Short: "List installed tools",
		Long:  "List currently installed tools",
		RunE: func(_ *cobra.Command, _ []string) error {
			return List()
		},
	}
	return installedCmd
}

func List() error {
	toolMap := tools.GetMap()

	fmt.Println("Currently installed tools:")
	for _, t := range toolMap {
		installed, err := t.Installed()
		if err != nil {
			return fmt.Errorf("failed to determine if '%s' has been installed: %w", t.GetName(), err)
		}
		if installed {
			installedVersion, err := t.InstalledVersion()
			if err != nil {
				return fmt.Errorf("failed to determine version for '%s': %w", t.GetName(), err)
			}
			fmt.Printf("- %s %s\n", t.GetName(), installedVersion)
		}
	}
	return nil
}
