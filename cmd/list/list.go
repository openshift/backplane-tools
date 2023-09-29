package list

import (
	"github.com/openshift/backplane-tools/cmd/list/available"
	"github.com/openshift/backplane-tools/cmd/list/installed"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use: "list",
		Args: cobra.NoArgs,
		Short: "List tools",
		Long: "List installed & available tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	listCmd.AddCommand(available.Cmd())
	listCmd.AddCommand(installed.Cmd())

	return listCmd
}
