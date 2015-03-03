package main

import (
	"os"
	"github.com/openshift/openshift-extras/diagnostics/cmd"
)

func main() {
	command := cmd.NewCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
