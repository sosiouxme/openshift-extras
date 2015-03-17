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

type logEntry struct {
	Message string // I feel certain we will want more fields at some point
}

type logMatcher struct { // regex for scanning log messages and interpreting them when found
	Regexp         *regexp.Regexp
	Level          log.Level
	Interpretation string // log at above level if it's simple
	KeepAfterMatch bool   // usually note only one log entry
	Interpret      func(  // run this to do its own logic
		env *types.Environment,
		entry *logEntry,
		matches []string,
	) bool // KeepAfterMatch?
}

type unitSpec struct {
	Name        string
	StartMatch  *regexp.Regexp // regex to look for in log messages indicating startup
	LogMatchers []logMatcher   // suspect log patterns to check for - checked in order
}

//
// -------- Things that feed into the diagnostics definitions -----------
// Search for Diagnostics for the actual diagnostics.

// Reusable log matchers:
var badImageTemplate = logMatcher{
	Regexp: regexp.MustCompile(`Unable to find an image for .* due to an error processing the format: %!v\\(MISSING\\)`),
	Level:  log.InfoLevel,
	Interpretation: `
This error indicates openshift was given the flag --images including an invalid format variable.
Valid formats can include (literally) ${component} and ${version}.
This could be a typo or you might be intending to hardcode something,
such as a version which should be specified as e.g. v3.0, not ${v3.0}.
Note that the --images flag may be supplied via the OpenShift master,
node, or "openshift ex registry/router" invocations and should usually
be the same for each.`,
}

// captures for logMatcher Interpret functions to store state between matches
var tlsClientErrorSeen map[string]bool

// Specify what units we can check and what to look for and say about it
var unitLogSpecs = []*unitSpec{
	&unitSpec{
		Name:       "openshift-master",
		StartMatch: regexp.MustCompile("Starting an OpenShift master"),
		LogMatchers: []logMatcher{
			badImageTemplate,
			logMatcher{
				Regexp:         regexp.MustCompile("Unable to decode an event from the watch stream: local error: unexpected message"),
				Level:          log.InfoLevel,
				Interpretation: "You can safely ignore this message.",
			},
			logMatcher{
				Regexp: regexp.MustCompile("HTTP probe error: Get .*/healthz: dial tcp .*:10250: connection refused"),
				Level:  log.InfoLevel,
				Interpretation: `
The OpenShift master does a health check on nodes that are defined in
its records, and this is the result when the node is not available yet.
Since the master records are typically created before the node is
available, this is not usually a problem, unless it continues in the
logs after the node is actually available.`,
			},
			logMatcher{
				Regexp: regexp.MustCompile("http: TLS handshake error from (\\S+): remote error: bad certificate"),
				Level:  log.WarnLevel,
				Interpret: func(env *types.Environment, entry *logEntry, matches []string) bool {
					client := matches[1]
					prelude := fmt.Sprintf("Found 'openshift-master' journald log message:\n  %s\n", entry.Message)
					if tlsClientErrorSeen == nil { // first time this message was seen
						tlsClientErrorSeen = map[string]bool{client: true}
						log.Warn(prelude + `
This error indicates that a client attempted to connect to the master
HTTPS API server but broke off the connection because the master's
certificate is not validated by a cerificate authority (CA) acceptable
to the client. There are a number of ways this can occur, some more
problematic than others.

At this time, the OpenShift master certificate is signed by a private CA
(created the first time the master runs) and clients should have a copy of
that CA certificate in order to validate connections to the master. Most
likely, either:
1. the master has generated a new CA (after the administrator deleted
   the old one) and the client has a copy of the old CA cert, or
2. the client hasn't been configured with a private CA at all (or the
   wrong one), or
3. the client is attempting to reach the master at a URL that isn't
   covered by the master's server certificate, e.g. a public-facing
   name or IP that isn't known to the master automatically; this may
   need to be specified with the --public-master flag on the master
   in order to generate a new server certificate including it.

Clients of the master may include users, nodes, and infrastructure
components running as containers. Check the "from" IP address in the
log message:
* If it is from a SDN IP, it is likely from an infrastructure
  component. Check pod logs and recreate it with the correct CA cert.
  Routers and registries won't work properly with the wrong CA.
* If it is from a node IP, the client is likely a node. Check the
  openshift-node and openshift-sdn-node logs and reconfigure with the
  correct CA cert. Nodes will be unable to create pods until this is
  corrected.
* If it is from an external IP, it is likely from a user (CLI, browser,
  etc.). osc and openshift clients should be configured with the correct
  CA cert; browsers can also add CA certs but it is usually easier
  to just have them accept the server certificate on the first visit
  (so this message may simply indicate that the master generated a new
  server certificate, e.g. to add a different --public-master, and a
  browser hasn't accepted it yet and is still attempting API calls;
  try logging out of the console and back in again).`)
					} else if !tlsClientErrorSeen[client] {
						tlsClientErrorSeen[client] = true
						log.Warn(prelude + `This message was diagnosed above, but for a different client address.`)
					} // else, it's a repeat, don't mention it
					return true // show once for every client failing to connect, not just the first
				},
			},
			logMatcher{
				Regexp:         regexp.MustCompile("user &{system:anonymous  [system:unauthenticated]} -> /api/v1beta1/services?namespace="),
				Level:          log.InfoLevel,
				Interpretation: ``,
			},
		},
	},
	&unitSpec{
		Name:        "openshift-sdn-master",
		StartMatch:  regexp.MustCompile("Starting OpenShift SDN Master"),
		LogMatchers: []logMatcher{},
	},
	&unitSpec{
		Name:       "openshift-node",
		StartMatch: regexp.MustCompile("Starting an OpenShift node"),
		LogMatchers: []logMatcher{
			badImageTemplate,
			logMatcher{
				Regexp: regexp.MustCompile("Unable to load services: Get (http\\S+/api/v1beta1/services\\?namespace=): (.+)"), // e.g. x509: certificate signed by unknown authority
				Level:  log.ErrorLevel,
				Interpretation: `
openshift-node could not connect to the OpenShift master API in order
to determine its responsibilities. This host will not function as a node
until this is resolved. Pods scheduled for this node will remain in
pending or unknown state forever.`,
			},
			logMatcher{
				Regexp: regexp.MustCompile(`Unable to load services: request.*403 Forbidden: Forbidden: "/api/v1beta1/services\?namespace=" denied by default`),
				Level:  log.ErrorLevel,
				Interpretation: `
openshift-node could not connect to the OpenShift master API to determine
its responsibilities because it lacks the proper credentials. Nodes
should specify a client certificate in order to identify themselves to
the master. This message typically means that either no client key/cert
was supplied, or it is not validated by the certificate authority (CA)
the master uses. You should supply a correct client key and certificate
to the .kubeconfig specified in /etc/sysconfig/openshift-node

This host will not function as a node until this is resolved. Pods
scheduled for this node will remain in pending or unknown state forever.`,
			},
		},
	},
	&unitSpec{
		Name:       "openshift-sdn-node",
		StartMatch: regexp.MustCompile("Starting OpenShift SDN node"),
		LogMatchers: []logMatcher{
			logMatcher{
				Regexp: regexp.MustCompile("Could not find an allocated subnet for this minion.*Waiting.."),
				Level:  log.WarnLevel,
				Interpretation: `
This warning occurs when openshift-sdn-node is trying to request the
SDN subnet it should be configured with according to openshift-sdn-master,
but either can't connect to it ("All the given peers are not reachable")
or has not yet been assigned a subnet ("Key not found").

This can just be a matter of waiting for the master to become fully
available and define a record for the node (aka "minion") to use,
and openshift-sdn-node will wait until that occurs, so the presence
of this message in the node log isn't necessarily a problem as
long as the SDN is actually working, but this message may help indicate
the problem if it is not working.

If the master is available and this node's record is defined and this
message persists, then it may be a sign of a different misconfiguration.
Unfortunately the message is not specific about why the connection failed.
Check MASTER_URL in /etc/sysconfig/openshift-sdn-node:
 * Is the protocol https? It should be http.
 * Can you reach the address and port from the node using curl?
   ("404 page not found" is correct response)`,
			},
		},
	},
	&unitSpec{
		Name:       "docker",
		StartMatch: regexp.MustCompile(`Starting Docker Application Container Engine.`), // RHEL Docker at least
		LogMatchers: []logMatcher{
			logMatcher{
				Regexp: regexp.MustCompile(`Usage: docker \\[OPTIONS\\] COMMAND`),
				Level:  log.ErrorLevel,
				Interpretation: `
This indicates that docker failed to parse its command line
successfully, so it just printed a standard usage message and exited.
Its command line is built from variables in /etc/sysconfig/docker
(which may be overridden by variables in /etc/sysconfig/openshift-sdn-node)
so check there for problems.

The OpenShift node will not work on this host until this is resolved.`,
			},
			logMatcher{ // generic error seen - do this last
				Regexp: regexp.MustCompile(`\\slevel="fatal"\\s`),
				Level:  log.ErrorLevel,
				Interpretation: `
This is not a known problem, but it is causing Docker to crash,
so the OpenShift node will not work on this host until it is resolved.`,
			},
		},
	},
	&unitSpec{
		Name:        "openvswitch",
		StartMatch:  regexp.MustCompile("Starting Open vSwitch"),
		LogMatchers: []logMatcher{},
	},
}

var systemdRelevant = func(env *types.Environment) (skip bool, reason string) {
	//return false, "" // for testing...
	if !env.HasSystemd {
		return true, "systemd is not present on this host"
	} else if env.OpenshiftPath == "" {
		return true, "`openshift` binary is not in the path on this host; we assume host is not a server"
	}
	return false, ""
}

//
// -------- The actual diagnostics definitions -----------
//

var Diagnostics = map[string]types.Diagnostic{

	"AnalyzeLogs": types.Diagnostic{
		Description: "Check for problems in systemd service logs since each service last started",
		Condition:   systemdRelevant,
		Run: func(env *types.Environment) {
			for _, unit := range unitLogSpecs {
				if svc := env.SystemdUnits[unit.Name]; svc.Enabled || svc.Active {
					log.Infof("Checking journalctl logs for '%s' service", unit.Name)
					matchLogsSinceLastStart(unit, env)
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
			// TODO: sdn+ovs will probably not be the only implementation - make this generic
			// Also, it's possible to run an all-in-one with no SDN
			unitRequiresUnit(u["openshift-node"], u["openshift-sdn-node"], `
The software-defined network (SDN) enables networking between
containers on different nodes. If it is not running, containers
on different nodes will not be able to connect to each other.`)
			unitRequiresUnit(u["openshift-sdn-master"], u["openshift-master"], `
The software-defined network (SDN) enables networking between containers
on different nodes, coordinated via openshift-sdn-master. It does not
make sense to run this service unless the host is operating as an
OpenShift master.`)
			// TODO: sdn+ovs will probably not be the only implementation - make this generic
			unitRequiresUnit(u["openshift-master"], u["openshift-sdn-master"], `
The software-defined network (SDN) enables networking between
containers on different nodes. If it is not running, containers
on different nodes will not be able to connect to each other.
openshift-sdn-master is required to provision the SDN subnets.`)
			unitRequiresUnit(u["openshift-sdn-node"], u["openvswitch"], `
The software-defined network (SDN) enables networking between
containers on different nodes. Containers will not be able to
connect to each other without the openvswitch service carrying
this traffic.`)
			unitRequiresUnit(u["openshift"], u["docker"], `OpenShift nodes use Docker to run containers.`)
			unitRequiresUnit(u["openshift"], u["iptables"], `
iptables is used by OpenShift nodes for container networking.
Connections to a container will fail without it.`)
			// sdn-node's dependency on node is a special case.
			// We do not need to enable node because sdn-note starts it for us.
			if u["openshift-sdn-node"].Active && !u["openshift-node"].Active {
				log.Error(`
systemd unit openshift-sdn-node is running but openshift-node is not.
Normally openshift-sdn-node starts openshift-node once initialized.
It is likely that openshift-node has crashed or been stopped.

An administrator can start openshift-node with:

  # systemctl start openshift-node

To ensure it is not repeatedly failing to run, check the status and logs with:

  # systemctl status openshift-node
  # journalctl -ru openshift-node `)
			}
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

func matchLogsSinceLastStart(unit *unitSpec, env *types.Environment) {
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
	entryTemplate := logEntry{Message: `json:"MESSAGE"`}
	matchCopy := append([]logMatcher(nil), unit.LogMatchers...) // make a copy, will remove matchers after they match something
	for lineReader.Scan() {                                     // each log entry is a line
		if len(matchCopy) == 0 { // if no rules remain to match
			break // don't waste time reading more log entries
		}
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
					prelude := fmt.Sprintf("Found '%s' journald log message:\n  %s\n", unit.Name, entry.Message)
					keep := match.KeepAfterMatch
					if match.Interpret != nil {
						keep = match.Interpret(env, &entry, strings)
					} else {
						log.Log(match.Level, prelude+match.Interpretation)
					}
					if !keep { // remove matcher once seen
						matchCopy = append(matchCopy[:index], matchCopy[index+1:]...)
					}
					break
				}
			}
		}
	}
}
