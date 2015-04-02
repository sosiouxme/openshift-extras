package cmd

import (
	"fmt"
	"github.com/openshift/openshift-extras/diagnostics/client"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/discovery"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/systemd"
	"github.com/openshift/openshift-extras/diagnostics/types"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/spf13/cobra"
	"os"
	"strings"
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
	cmd.AddCommand(newVersionCommand("version")) // print out diagnostics version
	osFlags := cmd.PersistentFlags()
	factory := osclientcmd.New(osFlags) // side effect: add standard flags for openshift client

	// Add flags separately from those inherited from the client
	diagFlags := &flags.Flags{OpenshiftFlags: osFlags, Diagnostics: make(flags.List, 0)}
	cmd.Flags().VarP(&diagFlags.Diagnostics, "diagnostics", "d", `comma-separated list of diagnostic names to run, e.g. "systemd.AnalyzeLogs"`)
	cmd.Flags().IntVarP(&diagFlags.LogLevel, "loglevel", "l", 2, "Level of output: 0 = Error, 1 = Warn, 2 = Info, 3 = Debug")
	cmd.Flags().StringVarP(&diagFlags.Format, "output", "o", "text", "Output format: text|json|yaml")
	cmd.Flags().StringVarP(&diagFlags.OpenshiftPath, "openshift", "", "", "Path to 'openshift' binary")
	cmd.Flags().StringVarP(&diagFlags.OscPath, "osc", "", "", "Path to 'osc' client binary")

	// set callback function for when this command is invoked:
	cmd.Run = func(c *cobra.Command, args []string) {
		log.SetLevel(diagFlags.LogLevel)
		c.SetOutput(os.Stdout)             // TODO: does this matter?
		log.SetLogFormat(diagFlags.Format) // note, ignore the error returned if format is unknown, just do text
		env := discovery.Run(diagFlags, factory)
		Diagnose(env, args)
		log.Summary()
		log.Finish()
	}

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

func Diagnose(env *types.Environment, args []string) {
	allDiags := map[string]map[string]types.Diagnostic{"client": client.Diagnostics, "systemd": systemd.Diagnostics}
	if list := env.Flags.Diagnostics; len(list) > 0 {
		// just run a specific (set of) diagnostic(s)
		for _, arg := range list {
			parts := strings.SplitN(arg, ".", 2)
			if len(parts) < 2 {
				log.Noticef("noDiag", `There is no such diagnostic "%s"`, arg)
				continue
			}
			area, name := parts[0], parts[1]
			if diagnostics, exists := allDiags[area]; !exists {
				log.Noticef("noDiag", `There is no such diagnostic "%s"`, arg)
			} else if diag, exists := diagnostics[name]; !exists {
				log.Noticef("noDiag", `There is no such diagnostic "%s"`, arg)
			} else {
				RunDiagnostic(area, name, diag, env)
			}
		}
	} else {
		// TODO: run all of these in parallel but ensure sane output
		for area, diagnostics := range allDiags {
			for name, diag := range diagnostics {
				RunDiagnostic(area, name, diag, env)
			}
		}
	}
}

func RunDiagnostic(area string, name string, diag types.Diagnostic, env *types.Environment) {
	defer func() {
		// recover from diagnostics that panic so others can still run
		if r := recover(); r != nil {
			log.Errorf("diagPanic", "Diagnostic '%s' crashed; this is usually a bug in either diagnostics or OpenShift. Stack trace:\n%+v", name, r)
		}
	}()
	if diag.Condition != nil {
		if skip, reason := diag.Condition(env); skip {
			if reason == "" {
				log.Noticem("diagSkip", log.Msg{"area": area, "name": name, "diag": diag.Description,
					"tmpl": "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}"})
			} else {
				log.Noticem("diagSkip", log.Msg{"area": area, "name": name, "diag": diag.Description, "reason": reason,
					"tmpl": "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}\nBecause: {{.reason}}"})
			}
			return
		}
	}
	log.Noticem("diagRun", log.Msg{"area": area, "name": name, "diag": diag.Description,
		"tmpl": "Running diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}"})
	diag.Run(env)
}
