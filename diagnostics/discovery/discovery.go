package discovery

import (
	"fmt"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	//XXX "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
  clientcmdlatest "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api/latest"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ----------------------------------------------------------
// Examine system and return findings in an Environment
func Run(f *flags.Flags) *types.Environment {
	env := &types.Environment{Flags: f}
	osDiscovery(env)
	execDiscovery(env)
	readKubeconfig(env)
	return env
}

// ----------------------------------------------------------
// Determine what we need to about the OS
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

// ----------------------------------------------------------
// Look for 'osc' and 'openshift' executables
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

// ----------------------------------------------------------
// Look for a specific executable and log what happens
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

// ----------------------------------------------------------
// Look in the path for an executable
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

// ----------------------------------------------------------
// Invoke executable's "version" command to determine version
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

/* ----------------------------------------------------------
Look for the .kubeconfig and read it in.

We are not going to attempt merging multiple files as the
actual client permits; most users will just be dealing with
a single kubeconfig, and we will be replacing all of that
with OpenShift-specific config soon enough.

We will look in the standard locations, alert the user to
what we find as we go along, and try to be helpful.
*/
func readKubeconfig(env *types.Environment) {
	file := findKubeconfig(env)
	if file == nil {
		log.Warn("No .kubeconfig read; using default config with OpenShift master at https://localhost:8443/")
	} else {
		defer file.Close()
		if buffer, err := ioutil.ReadAll(file); err != nil {
			log.Errorf("Unexpected error while reading .kubeconfig file (%s): %#v", file.Name(), err)
		} else {
			config := &clientcmdapi.Config{}
			if err := clientcmdlatest.Codec.DecodeInto(buffer, config); err != nil {
				// XXX: in post-0.4 rebase, becomes clientcmd.Load(buffer) - if we care
				log.Errorf("Error reading YAML from kubeconfig:\n%#v", err)
			} else {
				env.Kubeconfig = config
			}
		}
	}
}

// ----------------------------------------------------------
// Look for .kubeconfig in a number of possible locations
func findKubeconfig(env *types.Environment) (file *os.File){
	fPath := env.Flags.KubeconfigPath
	kcPath := os.Getenv("KUBECONFIG")
	adminPath1 := "/var/lib/openshift/openshift.certificates.d/admin/.kubeconfig" // enterprise
	adminPath2 := "/openshift.certificates.d/admin/.kubeconfig" // origin systemd
	adminWarningF := `
.kubeconfig was not available where expected; however, one exists at
  %s
which is a standard location where the master generates it.
Diagnostics will attempt to use this as your .kubeconfig now.
If this is what you want, you should copy it to a standard location
(your home directory, or the current directory), or you can set the
environment variable KUBECONFIG in your ~/.bash_profile:
  export KUBECONFIG=%s
If this is not what you want, you should obtain a .kubeconfig and
place it in a standard location.
`
 	if fPath != "" {
		// user specified intended path; don't keep looking if it isn't there.
		return openKubeconfig(fPath, fmt.Sprintf("-c specified that .kubeconfig should be at %s\n", fPath))
	} else if kcPath != "" {
		// $KUBECONFIG specified intended path; don't keep looking if it isn't there
		return openKubeconfig(kcPath, fmt.Sprintf("$KUBECONFIG specified that .kubeconfig should be at %s\n", kcPath))
	}
	// look for it in `pwd`
	path, _ := os.Getwd()
	if file = openKubeconfig(path+"/.kubeconfig", ""); file != nil {
		return file
	}
	// look for it in $HOME
	if file = openKubeconfig(os.Getenv("HOME")+"/.kubeconfig", ""); file != nil {
		return file
	}
	// look for it in auto-generated locations
	if file = openKubeconfig(adminPath1, ""); file != nil {
		log.Warnf(adminWarningF, adminPath1, adminPath1)
		return file
	}
	if file = openKubeconfig(adminPath2, ""); file != nil {
		log.Warnf(adminWarningF, adminPath2, adminPath2)
		return file
	}
	return file
}

// ----------------------------------------------------------
// Attempt to open file at path as .kubeconfig
// If there is a problem and errmsg is set, log an error
func openKubeconfig(path string, errmsg string) (file *os.File) {
	var err error
	if path != "" {
		if file, err = os.Open(path); err == nil {
			log.Infof("Reading .kubeconfig at %s", path)
		} else if errmsg == "" {
			log.Debugf("Could not read .kubeconfig at %s:\n%#v", path, err)
		} else if os.IsNotExist(err) {
			log.Error(errmsg + "but that file does not exist.")
		} else if os.IsPermission(err) {
			log.Error(errmsg + "but lack permission to read that file.")
		} else if err != nil {
			log.Errorf("%sbut there was an error opening it:\n%#v", errmsg, err)
		} // else it is open for reading
	}
	return file
}
