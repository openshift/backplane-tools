package main

import (
	"log"

	"github.com/openshift/backplane-tools/cmd/install"
	"github.com/openshift/backplane-tools/cmd/remove"
	"github.com/openshift/backplane-tools/cmd/upgrade"
	"github.com/spf13/cobra"
)

var cmd = cobra.Command{
	Use:   "backplane-tools",
	Short: "An OpenShift tool manager",
	Long:  "This applications manages the tools needed to interact with OpenShift clusters",
	RunE:  help,
}

func help(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

// Add subcommands
func init() {
	cmd.AddCommand(install.Cmd())
	cmd.AddCommand(upgrade.Cmd())
	cmd.AddCommand(remove.Cmd())
}

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
