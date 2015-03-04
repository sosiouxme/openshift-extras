package discovery

import (
	"fmt"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"os"
	"os/exec"
)

func Run(f *flags.Flags) *types.Environment {
	env := &types.Environment{Flags: f}
	osDiscovery(env)
	execDiscovery(env)
	return env
}

func osDiscovery(env *types.Environment) {
	//TODO: determine what we need to know about the OS
}

func execDiscovery(env *types.Environment) (err error) {
	log.Info("Searching for executables in path: " + os.Getenv("PATH")) //TODO for non-Linux OS
	env.Path4osc = findExecAndLog("osc")
	if env.Path4osc != "" {
		env.Version4osc, err = getExecVersion(env.Path4osc)
	}
	env.Path4openshift = findExecAndLog("openshift")
	if env.Path4openshift != "" {
		env.Version4openshift, err = getExecVersion(env.Path4openshift)
	}
	if env.Version4openshift.NonZero() && env.Version4osc.NonZero() && !env.Version4openshift.Eq(env.Version4osc) {
		log.Warnf("'openshift' version %#v does not match 'osc' version %#v; update or remove the lower version", env.Version4openshift, env.Version4osc)
	}
	return err
}

func findExecAndLog(cmd string) string {
	path := findExecFor(cmd)
	if path == "" {
		log.Warnf("No %v executable was found in your path", cmd)
	} else {
		log.Infof("Found %v at %v", cmd, path)
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

func getExecVersion(path string) (version types.Version, err error) {
	cmd := exec.Command(path, "version")
	var out []byte
	out, err = cmd.CombinedOutput()
	if err == nil {
		var name string
		var x, y, z int
		if _, err = fmt.Sscanf(string(out), "%s v%d.%d.%d", &name, &x, &y, &z); err == nil {
			version = types.Version{x, y, z}
			log.Infof("version of %s is %#v", name, version)
		} else {
			log.Errorf(`
Expected version output from '%s version'
Could not parse output received:
%v`, path, string(out))
		}
	} else {
		switch err.(type) {
		case *exec.Error:
			log.Errorf("error in executing '%v version': %v", path, err)
		case *exec.ExitError:
			log.Errorf(`
Executed '%v version' which exited with an error code.
This version is likely old or broken.
Error was '%v';
Output was:
%v`, path, err.Error(), log.LimitLines(string(out), 5))
		default:
			log.Errorf("executed '%v version' but an error occurred:\n%v\nOutput was:\n%v", path, err, string(out))
		}
	}

	return version, err
}
