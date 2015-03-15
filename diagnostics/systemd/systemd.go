package systemd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"io"
	"os/exec"
	"regexp"
)

type LogMatcher struct { // regex for scanning log messages and interpreting them when found
	Regexp         *regexp.Regexp
	Interpretation string                                       // if it's simple
	Interpret      func(logMsg string, matches []string) string // if logic is required
	Level          log.Level
}

type UnitSpec struct {
	Name        string
	StartMatch  *regexp.Regexp // regex to look for in log messages indicating startup
	LogMatchers []LogMatcher   // suspect log patterns to check for - checked in order
}

//
// -------- Things that feed into the diagnostics definitions -----------
//

// Reusable log matchers:
var badImageTemplate = LogMatcher{
	Regexp: regexp.MustCompile(`Unable to find an image for .* due to an error processing the format: %!v\\(MISSING\\)`),
	Interpretation: `
This error indicates openshift was given the flag --images including an invalid format variable.
Valid formats can include (literally) ${component} and ${version}
This could be a typo or you might be intending to hardcode something,
such as a version which should be specified as e.g. v3.0, not ${v3.0}`,
	Level: log.InfoLevel,
}

// Specify what units we can check and what to look for and say about it
var unitLogSpecs = []UnitSpec{
	UnitSpec{
		Name:       "openshift-master",
		StartMatch: regexp.MustCompile("Starting an OpenShift master"),
		LogMatchers: []LogMatcher{
			badImageTemplate,
			LogMatcher{
				Regexp:         regexp.MustCompile("Unable to decode an event from the watch stream: local error: unexpected message"),
				Interpretation: "You can safely ignore this message.",
				Level:          log.InfoLevel,
			},
		},
	},
	UnitSpec{
		Name:        "openshift-sdn-master",
		StartMatch:  regexp.MustCompile("Starting OpenShift SDN Master"),
		LogMatchers: []LogMatcher{},
	},
	UnitSpec{
		Name:       "openshift-node",
		StartMatch: regexp.MustCompile("Starting an OpenShift node"),
		LogMatchers: []LogMatcher{
			badImageTemplate,
		},
	},
	UnitSpec{
		Name:        "openshift-sdn-node",
		StartMatch:  regexp.MustCompile("Starting OpenShift SDN node"),
		LogMatchers: []LogMatcher{},
	},
	UnitSpec{
		Name:       "docker",
		StartMatch: regexp.MustCompile(`Starting Docker Application Container Engine.`), // RHEL Docker at least
		LogMatchers: []LogMatcher{
			LogMatcher{
				Regexp:         regexp.MustCompile(`Run 'docker COMMAND --help' for more information on a command.`),
				Interpretation: "This is not a known problem, but it is causing Docker to crash, so the OpenShift node will not work on this host until it is resolved.",
				Level:          log.ErrorLevel,
			},
			LogMatcher{ // generic error seen - do this last
				Regexp:         regexp.MustCompile(`\\slevel="fatal"\\s`),
				Interpretation: "This is not a known problem, but it is causing Docker to crash, so the OpenShift node will not work on this host until it is resolved.",
				Level:          log.ErrorLevel,
			},
		},
	},
	UnitSpec{
		Name:        "openvswitch",
		StartMatch:  regexp.MustCompile("Starting Open vSwitch"),
		LogMatchers: []LogMatcher{},
	},
}

var systemdRelevant = func(env *types.Environment) (skip bool, reason string) {
	//return false, "" // for testing...
	if !env.HasSystemd {
		return true, "systemd is not present on this host"
	} else if env.OpenshiftPath == "" {
		return true, "`openshift` binary not in the path on this host; for now, we assume host is not a server"
	}
	return false, ""
}

//
// -------- The actual diagnostics definitions -----------
//

var Diagnostics = map[string]types.Diagnostic{

	"AnalyzeLogs": types.Diagnostic{
		Description: "Check journald for known problems in relevant systemd unit logs",
		Condition:   systemdRelevant,
		Run: func(env *types.Environment) {
			for _, unit := range unitLogSpecs {
				if svc := env.SystemdUnits[unit.Name]; svc.Enabled || svc.Active {
					log.Infof("Checking journalctl logs for '%s' unit", unit.Name)
					matchLogsSinceLastStart(unit)
				}
			}
		},
	},

	"UnitStatus": types.Diagnostic{
		Description: "Check status for OpenShift-related systemd units",
		Condition:   systemdRelevant,
		Run: func(env *types.Environment) {
			u := env.SystemdUnits
			unitRequiresUnit(u["openshift-node"], u["iptables"], `
iptables is used by OpenShift nodes for container networking.
Connections to a container will fail without it.`)
			unitRequiresUnit(u["openshift-node"], u["docker"], `OpenShift nodes use Docker to run containers.`)
			unitRequiresUnit(u["openshift-sdn-node"], u["openshift-node"], `
The software-defined network (SDN) enables networking between
containers on different nodes. It does not make sense to run this
service unless the host is operating as an OpenShift node.`)
			unitRequiresUnit(u["openshift-sdn-master"], u["openshift-master"], `
The software-defined network (SDN) enables networking between containers
on different nodes, coordinated via openshift-sdn-master. It does not
make sense to run this service unless the host is operating as an
OpenShift master.`)
			unitRequiresUnit(u["openshift-sdn-node"], u["openvswitch"], `
The software-defined network (SDN) enables networking between
containers on different nodes. Containers will not be able to
connect to each other without the openvswitch service carrying
this traffic.`)
			unitRequiresUnit(u["openshift"], u["docker"], `OpenShift nodes use Docker to run containers.`)
			unitRequiresUnit(u["openshift"], u["iptables"], `
iptables is used by OpenShift nodes for container networking.
Connections to a container will fail without it.`)
			// Anything that is enabled but not running deserves notice
			for name, unit := range u {
				if unit.Enabled && !unit.Active {
					log.Errorf(`
The %s systemd unit is intended to start at boot but is not currently active.
An administrator can start the %[1]s unit with:
  # systemctl start %[1]s
To ensure it is not failing to run, check the status and logs with:
  # systemctl status %[1]s
  # journalctl -ru %[1]s`, name)
				}
			}
		},
	},
}

//
// -------- Functions used by the diagnostics -----------
//

func unitRequiresUnit(unit types.SystemdUnit, requires types.SystemdUnit, reason string) {
	if (unit.Active || unit.Enabled) && !requires.Exists {
		log.Errorf(`
systemd unit %s depends on unit %s, which is not loaded.
%s
An administrator probably needs to install the %[2]s unit with:

  # yum install %[2]s

If it is already installed, you may to reload the definition with:

  # systemctl reload %[2]s
		`, unit.Name, requires.Name, reason)
	} else if unit.Active && !requires.Active {
		log.Errorf(`
systemd unit %s is running but %s is not.
%s
An administrator can start the %[2]s unit with:

  # systemctl start %[2]s

To ensure it is not failing to run, check the status and logs with:

  # systemctl status %[2]s
  # journalctl -ru %[2]s
		`, unit.Name, requires.Name, reason)
	} else if unit.Enabled && !requires.Enabled {
		log.Warnf(`
systemd unit %s is enabled to run automatically at boot, but %s is not.
%s
An administrator can enable the %[2]s unit with:

  # systemctl enable %[2]s
		`, unit.Name, requires.Name, reason)
	}
}

func matchLogsSinceLastStart(unit UnitSpec) {
	cmd := exec.Command("journalctl", "-ru", unit.Name, "--output=json")
	// JSON comes out of journalctl one line per record
	lineReader, reader, err := func(cmd *exec.Cmd) (*bufio.Scanner, io.ReadCloser, error) {
		stdout, err := cmd.StdoutPipe()
		if err == nil {
			lineReader := bufio.NewScanner(stdout)
			if err = cmd.Start(); err == nil {
				return lineReader, stdout, nil
			}
		}
		return nil, nil, err
	}(cmd)
	if err != nil {
		log.Errorf("Diagnostics failed to query journalctl for the '%s' unit logs.\nThis should be very unusual, so please report the reason:\n(%T) %v", unit.Name, err, err)
		return
	}
	defer func() { // close out pipe once done reading
		reader.Close()
		cmd.Wait()
	}()
	var entryTemplate struct {
		Message string `json:"MESSAGE"` // I feel certain we will want more fields at some point
	}
	matchCopy := append([]LogMatcher(nil), unit.LogMatchers...) // make a copy, will remove matchers after they match something
	for lineReader.Scan() {                                     // each log entry is a line
		bytes, entry := lineReader.Bytes(), entryTemplate
		if err := json.Unmarshal(bytes, &entry); err != nil {
			log.Debugf("Couldn't read the JSON for this log message:\n%v\nGot error (%T) %v", bytes, err, err)
		} else {
			if unit.StartMatch.MatchString(entry.Message) {
				break // saw the log message where the unit started; done looking.
			}
			for index, match := range matchCopy { // match log message against provided matchers
				if strings := match.Regexp.FindStringSubmatch(entry.Message); strings != nil {
					// if matches: print interpretation, remove from matchCopy, and go on to next log entry
					prelude := fmt.Sprintf("Found  '%s' journald log message:\n  %s\n", unit.Name, entry.Message)
					if match.Interpret != nil {
						log.Log(match.Level, prelude+match.Interpret(string(bytes), strings))
					} else {
						log.Log(match.Level, prelude+match.Interpretation)
					}
					matchCopy = append(matchCopy[:index], matchCopy[index+1:]...) // remove matcher once seen
					break
				}
			}
		}
	}
}
