package main

import (
	"os"
	"path/filepath"

	"github.com/openshift/openshift-extras/diagnostics/pkg/diagnostics"
)

func main() {
	basename := filepath.Base(os.Args[0])
	command := diagnostics.CommandFor(basename)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
