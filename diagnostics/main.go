package main

import (
	"github.com/openshift/openshift-extras/diagnostics/cmd"
	"os"
)

func main() {
	command := cmd.NewCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
