package diagnostics

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const longDescription = `
OpenShift Diagnostics

This utility helps you understand and troubleshoot OpenShift v3.

    $ diagnostics

Note: This is a pre-alpha release of diagnostics and will change significantly.
`

// CommandFor returns the appropriate command for this base name,
// or the global OpenShift command
func CommandFor(basename string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "This utility helps you understand and troubleshoot OpenShift v3.",
		Long:  longDescription,
		Run: func(c *cobra.Command, args []string) {
			c.SetOutput(os.Stdout)
			c.Help()
		},
	}

	// cmd.SetUsageTemplate(templates.MainUsageTemplate())
	// cmd.SetHelpTemplate(templates.MainHelpTemplate())
	cmd.AddCommand(newVersionCommand("version"))

	return cmd
}

// newVersionCommand creates a command for displaying the version of this binary
func newVersionCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: "Display version",
		Run: func(c *cobra.Command, args []string) {
			fmt.Print("diagnostics alpha1 for openshift v3beta2\n")
		},
	}
}
