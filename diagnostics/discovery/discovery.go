package discovery

import (
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"os/exec"
	"runtime"
)

// ----------------------------------------------------------
// Examine system and return findings in an Environment
func Run(fl *flags.Flags) *types.Environment {
	log.Notice("discBegin", "Beginning discovery of environment")
	env := &types.Environment{Flags: fl, SystemdUnits: make(map[string]types.SystemdUnit)}
	operatingSystemDiscovery(env)
	clientDiscovery(env)
	discoverSystemd(env)
	readKubeconfig(env)
	return env
}

// ----------------------------------------------------------
// Determine what we need to about the OS
func operatingSystemDiscovery(env *types.Environment) {
	env.OS = runtime.GOOS
	if env.OS == "linux" {
		if _, err := exec.LookPath("systemctl"); err == nil {
			env.HasSystemd = true
		}
		if _, err := exec.LookPath("/bin/bash"); err == nil {
			env.HasBash = true
		}
	}
}
