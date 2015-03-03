package discovery

import (
	"fmt"
  "os/exec"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/log"
)

type Environment struct {
  OS string   // "Linux / Windows / Mac"
  HasSystemd bool
  Path4osc string
  Version4osc string
  Path4openshift string
  Version4openshift string
  Path4kubeconfig string
	Flags *flags.Flags
}

func Run(f *flags.Flags) (*Environment) {
  env := &Environment{ Flags: f}
	osDiscovery(env)
	execDiscovery(env)
  return env
}

func osDiscovery(env *Environment) {
  //TODO: determine what we need to know about the OS
}

func execDiscovery(env *Environment) (err error) {
	log.Info("Searching for executables in path: ")
  env.Path4osc = findExecAndLog("osc")
  if env.Path4osc != "" {
		env.Version4osc, err = getExecVersion(env.Path4osc)
	}
  env.Path4openshift = findExecAndLog("openshift")
  if env.Path4openshift != "" {
		env.Version4openshift, err = getExecVersion(env.Path4openshift)
	}
	return err
}

func findExecAndLog(cmd string) string {
	path := findExecFor(cmd)
  if path == "" {
		log.Info(fmt.Sprintf("No %v executable was found in your path", cmd))
	} else {
    log.Info(fmt.Sprintf("Found %v at %v", cmd, path))
  }
	return path
}

func findExecFor(cmd string) string {
  path, err := exec.LookPath(cmd)
  if err == nil {
    return path
  }
	// TODO: if windows...
  path, err = exec.LookPath(cmd + ".exe")
  if err == nil {
    return path
  }
  return ""
}

func getExecVersion(path string) (version string, err error)  {

  return version, err
}
