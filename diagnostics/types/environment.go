package types

import (
  "github.com/openshift/openshift-extras/diagnostics/cmd/flags"
)

type Environment struct {
  OS string   // "Linux / Windows / Mac"
  HasSystemd bool
	HasBash bool
  Path4osc string
  Version4osc Version

  Path4openshift string
  Version4openshift Version
  Path4kubeconfig string
	Flags *flags.Flags
}
