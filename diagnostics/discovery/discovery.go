package discovery

import (
	"fmt"
  "os/exec"
	"github.com/openshift/openshift-extras/diagnostics/log"
)

type Environment struct {
  OS string   // "Linux / Windows / Mac"
  HasSystemd bool
  Path4osc string
  Path4openshift string
  Path4kubeconfig string
}

func Discover() *Environment {
  env := &Environment{}
  env.Path4osc = findExecFor("osc")
  env.Path4openshift = findExecFor("openshift")
  if env.Path4osc != "" {
    log.Info(fmt.Sprintf("Found osc at %v", env.Path4osc))
  }
  return env
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
  log.Warn(fmt.Sprintf("Could not find %v", cmd))
  return ""
}
