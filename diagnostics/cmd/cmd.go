package cmd

import (
	"fmt"
	"os"

	"github.com/openshift/openshift-extras/diagnostics/client"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/discovery"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/systemd"
	"github.com/openshift/openshift-extras/diagnostics/types"
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
	// callback function for when this command is invoked
	cmd.Run = func(c *cobra.Command, args []string) {
		log.SetLevel(diagFlags.LogLevel)
		c.SetOutput(os.Stdout)             // TODO: does this matter?
		log.SetLogFormat(diagFlags.Format) // ignore error
		env := discovery.Run(&diagFlags)
		// set up openshift/kube client objects
		env.Command = c
		env.Factory = factory
		env.OsClient, env.KubeClient, _ = factory.Clients(c)
		// run the diagnostics
		Diagnose(env)
		// summarize...
		log.Summary()
		log.Finish()
	}

	cmd.AddCommand(newVersionCommand("version"))
	// Add flags separately from those inherited from the client
	cmd.Flags().IntVarP(&diagFlags.LogLevel, "loglevel", "l", 2, "Level of output: 0 = Error, 1 = Warn, 2 = Info, 3 = Debug")
	cmd.Flags().StringVarP(&diagFlags.Format, "output", "o", "text", "Output format: text|json|yaml")
	cmd.Flags().StringVarP(&diagFlags.OpenshiftPath, "openshift", "", "", "Path to 'openshift' binary")
	cmd.Flags().StringVarP(&diagFlags.OscPath, "osc", "", "", "Path to 'osc' client binary")

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

func Diagnose(env *types.Environment) {
	allDiags := map[string]map[string]types.Diagnostic{"client": client.Diagnostics, "systemd": systemd.Diagnostics}
	// TODO: run all of these in parallel but ensure sane output
	// TODO: recover from diagnostics that panic so others can still run
	// TODO: just run a specific (set of) diagnostic(s)
	for area, diagnostics := range allDiags {
		for name, d := range diagnostics {
			if d.Condition != nil {
				if skip, reason := d.Condition(env); skip {
					if reason == "" {
						log.Noticem("diagSkip", log.Msg{"area": area, "name": name, "diag": d.Description,
							"tmpl": "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}"})
					} else {
						log.Noticem("diagSkip", log.Msg{"area": area, "name": name, "diag": d.Description, "reason": reason,
							"tmpl": "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}\nBecause: {{.reason}}"})
					}
					continue
				}
			}
			log.Noticem("diagRun", log.Msg{"area": area, "name": name, "diag": d.Description,
				"tmpl": "Running diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}"})
			d.Run(env)
		}
	}
}
