package discovery

import (
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/spf13/cobra"
	"os/exec"
	"runtime"
)

// ----------------------------------------------------------
// Examine system and return findings in an Environment
func Run(fl *flags.Flags, f *osclientcmd.Factory, c *cobra.Command) *types.Environment {
	log.Notice("discBegin", "Beginning discovery of environment")
	env := types.NewEnvironment(fl, f, c)
	if config, err := f.OpenShiftClientConfig.RawConfig(); err != nil {
		log.Errorf("discCCstart", "Could not read client config: (%T) %[1]v", err)
	} else {
		env.OsConfig = &config
		env.FactoryForContext[config.CurrentContext] = f
	}
	// set up openshift/kube client objects; side effect, finalize config
	env.OsClient, env.KubeClient, _ = f.Clients() // TODO no real reason to store these specially, rip out
	// run discovery
	operatingSystemDiscovery(env)
	clientDiscovery(env)
	discoverSystemd(env)
	readClientConfigFiles(env) // so user knows where config is coming from (or not)
	configClient(env)
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
