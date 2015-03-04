package cmd

import (
	"fmt"
	"os"

	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/discovery"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/spf13/cobra"
)

const longDescription = `
OpenShift Diagnostics

This utility helps you understand and troubleshoot OpenShift v3.
It requires a client binary ('osc' or 'openshift') to perform diagnostics.

    $ diagnostics

Note: This is a pre-alpha version of diagnostics and will change significantly.
`

func NewCommand() *cobra.Command {
	f := flags.Flags{}
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "This utility helps you understand and troubleshoot OpenShift v3.",
		Long:  longDescription,
		Run: func(c *cobra.Command, args []string) {
			log.SetLevel(f.LogLevel)
			c.SetOutput(os.Stdout)
			discovery.Run(&f)
		},
	}

	cmd.AddCommand(newVersionCommand("version"))
	cmd.Flags().IntVarP(&f.LogLevel, "loglevel", "l", 2, "Level of output: 0 = Error, 1 = Warn, 2 = Info, 3 = Debug")
	cmd.Flags().StringVarP(&f.OpenshiftPath, "openshift", "O", "", "Path to 'openshift' binary")
	cmd.Flags().StringVarP(&f.OscPath, "osc", "o", "", "Path to 'osc' client binary")
	cmd.Flags().StringVarP(&f.KubeconfigPath, "config", "c", "", "Path to '.kubeconfig' client config file")

	return cmd
}

// newVersionCommand creates a subcommand for displaying the version of this binary
func newVersionCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: "Display version",
		Run: func(c *cobra.Command, args []string) {
			fmt.Print("diagnostics alpha1 for openshift v3beta2\n")
		},
	}
}
