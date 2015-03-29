package discovery

import (
	"fmt"
	//XXX "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// ----------------------------------------------------------
// Look for 'osc' and 'openshift' executables
func clientDiscovery(env *types.Environment) (err error) {
	log.Debug("discSearchExec", "Searching for executables in path:\n  "+strings.Join(filepath.SplitList(os.Getenv("PATH")), "\n  ")) //TODO for non-Linux OS
	env.OscPath = findExecAndLog("osc", env, env.Flags.OscPath)
	if env.OscPath != "" {
		env.OscVersion, err = getExecVersion(env.OscPath)
	}
	env.OpenshiftPath = findExecAndLog("openshift", env, env.Flags.OpenshiftPath)
	if env.OpenshiftPath != "" {
		env.OpenshiftVersion, err = getExecVersion(env.OpenshiftPath)
	}
	if env.OpenshiftVersion.NonZero() && env.OscVersion.NonZero() && !env.OpenshiftVersion.Eq(env.OscVersion) {
		log.Warnm("discVersionMM", log.Msg{"osV": env.OpenshiftVersion.GoString(), "oscV": env.OscVersion.GoString(),
			"text": fmt.Sprintf("'openshift' version %#v does not match 'osc' version %#v; update or remove the lower version", env.OpenshiftVersion, env.OscVersion)})
	}
	return err
}

// ----------------------------------------------------------
// Look for a specific executable and log what happens
func findExecAndLog(cmd string, env *types.Environment, pathflag string) string {
	if pathflag != "" { // look for it where the user said it would be
		if filepath.Base(pathflag) != cmd {
			log.Errorm("discExecFlag", log.Msg{"command": cmd, "path": pathflag, "tmpl": `
You specified that '{{.command}}' should be found at:
  {{.path}}
  but that file has the wrong name. The file name determines available functionality and must match.`})
		} else if _, err := exec.LookPath(pathflag); err == nil {
			log.Infom("discExecFound", log.Msg{"command": cmd, "path": pathflag,
				"tmpl": "Specified '{{.command}}' is executable at {{.path}}"})
			return pathflag
		} else if _, err := os.Stat(pathflag); os.IsNotExist(err) {
			log.Errorm("discExecNoExist", log.Msg{"command": cmd, "path": pathflag,
				"tmpl": "You specified that '{{.command}}' should be at {{.path}}\nbut that file does not exist."})
		} else {
			log.Errorm("discExecNot", log.Msg{"command": cmd, "path": pathflag,
				"tmpl": "You specified that '{{.command}}' should be at {{.path}}\nbut that file is not executable."})
		}
	} else { // look for it in the path
		path := findExecFor(cmd)
		if path == "" {
			log.Warnm("discExecNoPath", log.Msg{"command": cmd, "tmpl": "No '{{.command}}' executable was found in your path"})
		} else {
			log.Infom("discExecFound", log.Msg{"command": cmd, "path": path, "tmpl": "Found '{{.command}}' at {{.path}}"})
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
		if scanned, err := fmt.Sscanf(string(out), "%s v%d.%d.%d", &name, &x, &y, &z); scanned > 1 {
			version = types.Version{x, y, z}
			log.Infom("discVersion", log.Msg{"tmpl": "version of {{.command}} is {{.version}}", "command": name, "version": version.GoString()})
		} else {
			log.Errorf("discVersErr", `
Expected version output from '%s version'
Could not parse output received:
%v
Error was: %#v`, path, string(out), err)
		}
	} else {
		switch err.(type) {
		case *exec.Error:
			log.Errorf("discVersErr", "error in executing '%v version': %v", path, err)
		case *exec.ExitError:
			log.Errorf("discVersErr", `
Executed '%v version' which exited with an error code.
This version is likely old or broken.
Error was '%v';
Output was:
%v`, path, err.Error(), log.LimitLines(string(out), 5))
		default:
			log.Errorf("discVersErr", "executed '%v version' but an error occurred:\n%v\nOutput was:\n%v", path, err, string(out))
		}
	}
	return version, err
}

// ----------------------------------------------------------
// Determine what systemd units are relevant, if any
// Run after determining whether systemd and openshift are present.
func discoverSystemd(env *types.Environment) {
	if env.OpenshiftPath == "" || !env.HasSystemd {
		/* If no openshift executable, for now we assume OpenShift is not running here,
		 * in which case we don't much care about systemd services.
		 * TODO: in the future, OpenShift could be running in a docker container,
		 * and could depend on services running separately (kubernetes, etcd).
		 * Handle this gracefully, as well as `openshift start` processes
		 * running outside systemd */
		return
	}
	if env.HasSystemd { // discover units we care about
		for _, name := range []string{"openshift", "openshift-master", "openshift-node", "openshift-sdn-master", "openshift-sdn-node", "docker", "openvswitch", "iptables", "etcd", "kubernetes"} {
			if unit := discoverSystemdUnit(name); unit.Exists {
				env.SystemdUnits[name] = unit
				log.Debugm("discUnit", log.Msg{"tmpl": "Saw systemd unit {{.unit}}", "unit": name})
			}
		}
	}
	log.Debugf("discUnits", "%v", env.SystemdUnits)
}

func discoverSystemdUnit(name string) types.SystemdUnit {
	unit := types.SystemdUnit{Name: name, Exists: false}
	if output, err := exec.Command("systemctl", "show", name).Output(); err != nil {
		log.Errorm("discCtlErr", log.Msg{"tmpl": "Error running `systemctl show {{.unit}}`: {{.error}}\nCannot analyze systemd units.", "unit": name, "error": err.Error()})
	} else {
		attr := make(map[string]string)
		for _, line := range strings.Split(string(output), "\n") {
			elements := strings.SplitN(line, "=", 2) // Looking for "Foo=Bar" settings
			if len(elements) == 2 {                  // found that, record it...
				attr[elements[0]] = elements[1]
			}
		}
		if val := attr["LoadState"]; val != "loaded" {
			log.Debugm("discUnitENoExist", log.Msg{"tmpl": "systemd unit '{{.unit}}' does not exist. LoadState is '{{.state}}'", "unit": name, "state": val})
			return unit // doesn't exist - leave everything blank
		} else {
			unit.Exists = true
		}
		if val := attr["UnitFileState"]; val == "enabled" {
			log.Debugm("discUnitEnabled", log.Msg{"tmpl": "systemd unit '{{.unit}}' is enabled - it will start automatically at boot.", "unit": name})
			unit.Enabled = true
		} else {
			log.Debugm("discUnitNoEnable", log.Msg{"tmpl": "systemd unit '{{.unit}}' is not enabled - it does not start automatically at boot. UnitFileState is '{{.state}}'", "unit": name, "state": val})
		}
		if val := attr["ActiveState"]; val == "active" {
			log.Debugm("discUnitActive", log.Msg{"tmpl": "systemd unit '{{.unit}}' is currently running", "unit": name})
			unit.Active = true
		} else {
			log.Debugm("discUnitNoActive", log.Msg{"unit": name, "state": val, "exit": unit.ExitStatus,
				"tmpl": "systemd unit '{{.unit}}' is not currently running. ActiveState is '{{.state}}'; exit code was {{.exit}}."})
		}
		fmt.Sscanf(attr["StatusErrno"], "%d", &unit.ExitStatus) // ignore errors...
	}
	return unit
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
		log.Warn("discNoKC", "No .kubeconfig read; default OpenShift config will be used, which is likely not what you want.")
	} else {
		defer file.Close()
		if buffer, err := ioutil.ReadAll(file); err != nil {
			log.Errorf("discKCReadErr", "Unexpected error while reading .kubeconfig file (%s): %v", file.Name(), err)
		} else {
			if config, err := clientcmd.Load(buffer); err != nil {
				log.Errorf("discKCYamlErr", `Error reading YAML from kubeconfig file (%s):
  %v
This file may have been truncated or mis-edited.
Please fix or get a new .kubeconfig`, file.Name(), err)
			} else {
				/* Note, we're not actually going to use this config,
				 * because it's too hard to turn it into something that's useful.
				 * Instead, we'll defer to the openshift client code to assimilate
				 * flags, env vars, and the potential hierarchy of kubeconfig files
				 * into an actual configuration that the client uses.
				 */
				env.Kubeconfig = config
				log.Infom("discKCRead", log.Msg{"tmpl": `Successfully read a .kubeconfig file at '{{.path}}';
be aware that the actual configuration used later may be different
due to environment variables, flags, and other .kubeconfig files
being merged together.`, "path": file.Name()})
			}
		}
	}
}

// ----------------------------------------------------------
// Look for .kubeconfig in a number of possible locations
func findKubeconfig(env *types.Environment) (file *os.File) {
	fPath := env.Flags.OpenshiftFlags.Lookup("config").Value.String()
	kcPath := os.Getenv("KUBECONFIG")
	adminPath1 := "/var/lib/openshift/openshift.certificates.d/admin/.kubeconfig" // enterprise
	adminPath2 := "/openshift.certificates.d/admin/.kubeconfig"                   // origin systemd
	adminWarningF := `
.kubeconfig was not available where expected; however, one exists at
  %s
which is a standard location where the master generates it.
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
	if file = openKubeconfig(os.Getenv("HOME")+"/.kube/.kubeconfig", ""); file != nil {
		return file
	}
	// look for it in auto-generated locations when not found properly
	if file = openKubeconfig(adminPath1, ""); file != nil {
		file.Close()
		file = nil
		log.Warnf("discKCautoPath", adminWarningF, adminPath1, adminPath1)
	}
	if file = openKubeconfig(adminPath2, ""); file != nil {
		file.Close()
		file = nil
		log.Warnf("discKCautoPath", adminWarningF, adminPath2, adminPath2)
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
			log.Infom("discOpenKC", log.Msg{"tmpl": "Reading .kubeconfig at {{.path}}", "path": path})
		} else if errmsg == "" {
			log.Debugf("discOpenKCNo", "Could not read .kubeconfig at %s:\n%#v", path, err)
		} else if os.IsNotExist(err) {
			log.Error("discOpenKCNoExist", errmsg+"but that file does not exist.")
		} else if os.IsPermission(err) {
			log.Error("discOpenKCNoPerm", errmsg+"but lack permission to read that file.")
		} else if err != nil {
			log.Errorf("discOpenKCErr", "%sbut there was an error opening it:\n%#v", errmsg, err)
		} // else it is open for reading
	}
	return file
}
