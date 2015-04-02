package discovery

import (
	"fmt"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"os/exec"
	"strings"
)

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