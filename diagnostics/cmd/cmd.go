package cmd

import (
	"fmt"
	"os"

	"github.com/openshift/openshift-extras/diagnostics/client"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/discovery"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
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
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "This utility helps you understand and troubleshoot OpenShift v3.",
		Long:  longDescription,
	}
	osFlags := cmd.PersistentFlags()
	diagFlags := flags.Flags{OpenshiftFlags: osFlags}
	factory := clientcmd.New(osFlags)
	cmd.Run = func(c *cobra.Command, args []string) {
		log.SetLevel(diagFlags.LogLevel)
		c.SetOutput(os.Stdout)
		env := discovery.Run(&diagFlags)
		env.Command = c
		env.Factory = factory
		env.OsClient, env.KubeClient, _ = factory.Clients(c)
		client.Diagnose(env)
		log.Summary()
	}

	cmd.AddCommand(newVersionCommand("version"))
	// Add flags separately from those inherited from the client
	cmd.Flags().IntVarP(&diagFlags.LogLevel, "loglevel", "l", 2, "Level of output: 0 = Error, 1 = Warn, 2 = Info, 3 = Debug")
	cmd.Flags().StringVarP(&diagFlags.OpenshiftPath, "openshift", "O", "", "Path to 'openshift' binary")
	cmd.Flags().StringVarP(&diagFlags.OscPath, "osc", "o", "", "Path to 'osc' client binary")
	//cmd.Flags().StringVarP(&f.KubeconfigPath, "config", "c", "", "Path to '.kubeconfig' client config file")

	return cmd
}

// newVersionCommand creates a subcommand for displaying the version of this binary
func newVersionCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: "Display version of the diagnostics tool",
		Run: func(c *cobra.Command, args []string) {
			fmt.Print("diagnostics alpha1 for openshift v3beta2\n")
		},
	}
}
