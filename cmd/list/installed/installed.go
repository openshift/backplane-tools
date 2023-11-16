package installed

import (
	"fmt"

	"github.com/openshift/backplane-tools/pkg/tool"
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
	toolMap := tool.GetMap()
	installDir, err := tool.InstallDir()
	if err != nil {
		return fmt.Errorf("failed to determine installation directory: %w", err)
	}

	fmt.Println("Currently installed tools:")
	for _, t := range toolMap {
		installed, err := t.Installed(installDir)
		if err != nil {
			return fmt.Errorf("failed to determine if '%s' has been installed: %w", t.Name(), err)
		}
		if installed {
			fmt.Printf("- %s\n", t.Name())
		}
	}
	return nil
}
