package discovery

import (
	"fmt"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Run(f *flags.Flags) *types.Environment {
	env := &types.Environment{Flags: f}
	osDiscovery(env)
	execDiscovery(env)
	return env
}

func osDiscovery(env *types.Environment) {
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

func execDiscovery(env *types.Environment) (err error) {
	log.Debug("Searching for executables in path:\n  " + strings.Join(filepath.SplitList(os.Getenv("PATH")), "\n  ")) //TODO for non-Linux OS
	env.OscPath = findExecAndLog("osc", env, env.Flags.OscPath)
	if env.OscPath != "" {
		env.OscVersion, err = getExecVersion(env.OscPath)
	}
	env.OpenshiftPath = findExecAndLog("openshift", env, env.Flags.OpenshiftPath)
	if env.OpenshiftPath != "" {
		env.OpenshiftVersion, err = getExecVersion(env.OpenshiftPath)
	}
	if env.OpenshiftVersion.NonZero() && env.OscVersion.NonZero() && !env.OpenshiftVersion.Eq(env.OscVersion) {
		log.Warnf("'openshift' version %#v does not match 'osc' version %#v; update or remove the lower version", env.OpenshiftVersion, env.OscVersion)
	}
	return err
}

func findExecAndLog(cmd string, env *types.Environment, pathflag string) string {
	if pathflag != "" { // look for it where the user said it would be
		if filepath.Base(pathflag) != cmd {
			log.Errorf(`
You specified that '%s' should be found at:
  %s
but that file has the wrong name. The file name determines available functionality and must match.`, cmd, pathflag)
		} else if _, err := exec.LookPath(pathflag); err == nil {
			log.Infof("Specified '%v' is executable at %v", cmd, pathflag)
			return pathflag
		} else if _, err := os.Stat(pathflag); os.IsNotExist(err) {
			log.Errorf("You specified that '%s' should be at %s\nbut that file does not exist.", cmd, pathflag)
		} else {
			log.Errorf("You specified that '%s' should be at %s\nbut that file is not executable.", cmd, pathflag)
		}
	} else { // look for it in the path
		path := findExecFor(cmd)
		if path == "" {
			log.Warnf("No '%v' executable was found in your path", cmd)
		} else {
			log.Infof("Found '%v' at %v", cmd, path)
			return path
		}
	}
	return ""
}

func findExecFor(cmd string) string {
	path, err := exec.LookPath(cmd)
	if err == nil {
		return path
	}
	if runtime.GOOS == "windows" {
		path, err = exec.LookPath(cmd + ".exe")
		if err == nil {
			return path
		}
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
