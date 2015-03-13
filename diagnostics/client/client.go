package client

import (
	"fmt"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/resource"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"regexp"
	"strings"
)

var Diagnostics = map[string]types.Diagnostic{
	"KubeconfigContexts": types.Diagnostic{
		Description: "Test that kubeconfig contexts and current context are ok",
		Condition: func(env *types.Environment) (skip bool, reason string) {
			if env.Kubeconfig == nil {
				return true, "There is no .kubeconfig file"
			}
			return false, ""
		},
		Run: func(env *types.Environment) {
			kc := env.Kubeconfig
			log.Info("Testing server configuration(s) from kubeconfig")
			cc := kc.CurrentContext
			ccResult, ccSuccess := "", false
			for context, _ := range kc.Contexts {
				result, success := TestContext(context, kc)
				result = fmt.Sprintf("For kubeconfig context '%s':%s", context, result)
				if context == cc {
					ccResult, ccSuccess = result, success
				} else if success {
					log.Info(result)
				} else {
					log.Warn(result)
				}
				// TODO: actually test whether these contexts are usable
			}
			if _, exists := kc.Contexts[cc]; exists {
				ccResult = fmt.Sprintf(`
The current context from kubeconfig is '%s'
This will be used by default to contact your OpenShift server.
%s`, cc, ccResult)
				if ccSuccess {
					log.Infof(ccResult)
				} else {
					log.Error(ccResult)
				}
			} else { // context does not exist
				log.Errorf(`
Your kubeconfig specifies a current context of '%s'
which is not defined; it is likely that a mistake was introduced while
manually editing your kubeconfig. If this is a simple typo, you may be
able to fix it manually.
The OpenShift master creates a fresh kubeconfig when it is started; it may be
useful to use this as a base if available.`, cc)
			}
		},
	},
	"ContactMaster": types.Diagnostic{
		Description: "Test contacting the OpenShift master",
		Run: func(env *types.Environment) {
			mapper, typer := env.Factory.Object(env.Command)
			_, err := resource.NewBuilder(mapper, typer, env.Factory.ClientMapperForCommand(env.Command)).
				ResourceTypeOrNameArgs("projects").
				Latest().
				Do().Object()
			if err != nil {
				noResolveRx := regexp.MustCompile("dial tcp: lookup (\\S+): no such host")
				unknownCaMsg := "x509: certificate signed by unknown authority"
				unneededCaMsg := "specifying a root certificates file with the insecure flag is not allowed"
				invalidCertNameRx := regexp.MustCompile("x509: certificate is valid for (\\S+, )+not (\\S+)")
				connRefusedRx := regexp.MustCompile("dial tcp (\\S+): connection refused")
				connTimeoutRx := regexp.MustCompile("dial tcp (\\S+): connection timed out")
				malformedHTTPMsg := "malformed HTTP response"
				malformedTLSMsg := "tls: oversized record received with length"

				// interpret the error message for mere mortals
				msg := err.Error()
				var reason string
				switch {
				case noResolveRx.MatchString(msg):
					reason = `
This usually means that the hostname does not resolve to an IP.
Hostnames should usually be resolved via an /etc/hosts file or DNS.
Ensure that the hostname resolves correctly from your host before proceeding.
Of course, you could also simply have the wrong hostname specified.
`
				case strings.Contains(msg, unknownCaMsg):
					reason = `
This means that we cannot validate the certificate in use by the
OpenShift API server, so we cannot securely communicate with it.
Connections could be intercepted and your credentials stolen.

Since the server certificate we see when connecting is not validated
by public certificate authorities (CAs), you probably need to specify a
certificate from a private CA to validate the connection.

You may be specifying the wrong CA cert, or none, or there could
actually be a man-in-the-middle attempting to intercept your
connection.  If you are unconcerned about any of this, you can add the
--insecure-skip-tls-verify flag to bypass secure (TLS) verification,
but this is risky and should not be necessary.
** Connections could be intercepted and your credentials stolen. **
`
				case strings.Contains(msg, unneededCaMsg):
					reason = `
This means that for client connections to the OpenShift API server, you
(or your kubeconfig) specified both a validating certificate authority
and that the client should bypass connection security validation.

This is not allowed because it is likely to be a mistake.

If you want to use --insecure-skip-tls-verify to bypass security (which
is usually a bad idea anyway), then you need to also clear the CA cert
from your command line options or kubeconfig file(s). Of course, it
would be far better to obtain and use a correct CA cert.
`
				case invalidCertNameRx.MatchString(msg):
					match := invalidCertNameRx.FindStringSubmatch(msg)
					serverHost := match[len(match)-1]
					reason = fmt.Sprintf(`
This means that the certificate in use by the OpenShift API server
(master) does not match the hostname by which you are addressing it:
  %s
so a secure connection is not allowed. In theory, this *could* mean that
someone is intercepting your connection and presenting a certificate
that is valid but for a different server, which is why secure validation
fails in this case.

However, the most likely explanation is that the server certificate
needs to be updated to include the name you are using to reach it.

If the OpenShift server is generating its own certificates (which
is default), then the --public-master flag on the OpenShift master is
usually the easiest way to do this. If you need something more complicated
(for instance, multiple public addresses for the API, or your own CA),
then you will need to custom-generate the server certificate with the
right names yourself.

If you are unconcerned about any of this, you can add the
--insecure-skip-tls-verify flag to bypass secure (TLS) verification,
but this is risky and should not be necessary.
** Connections could be intercepted and your credentials stolen. **
`, serverHost)
				case connRefusedRx.MatchString(msg):
					reason = fmt.Sprintf(`
This means that when we tried to connect to the OpenShift API
server (master), we reached the host, but nothing accepted the port
connection. This could mean that the OpenShift master is stopped, or
that a firewall or security policy is blocking access at that port.

You will not be able to connect or do anything at all with OpenShift
until this server problem is resolved or you specify a corrected
server address.
`)
				case connTimeoutRx.MatchString(msg):
					reason = fmt.Sprintf(`
This means that when we tried to connect to the OpenShift API server
(master), we could not reach the host at all.
* You may have specified the wrong host address.
* This could mean the host is completely unavailable (down).
* This could indicate a routing problem or a firewall that simply
  drops requests rather than responding by reseting the connection.
* It does not generally mean that DNS name resolution failed (which
  would be a different error) though the problem could be that it
  gave the wrong address.
`)
				case strings.Contains(msg, malformedHTTPMsg):
					reason = fmt.Sprintf(`
This means that when we tried to connect to the OpenShift API server
(master) with a plain HTTP connection, the server did not speak
HTTP back to us. The most common explanation is that a secure server
is listening but you specified an http: connection instead of https:.
There could also be another service listening at the intended port
speaking some other protocol entirely.

You will not be able to connect or do anything at all with OpenShift
until this server problem is resolved or you specify a corrected
server address.
`)
				case strings.Contains(msg, malformedTLSMsg):
					reason = fmt.Sprintf(`
This means that when we tried to connect to the OpenShift API server
(master) with a secure HTTPS connection, the server did not speak
HTTPS back to us. The most common explanation is that the server
listening at that port is not the secure server you expected - it
may be a non-secure HTTP server or the wrong service may be
listening there, or you may have specified an incorrect port.

You will not be able to connect or do anything at all with OpenShift
until this server problem is resolved or you specify a corrected
server address.
`)
				default:
					reason = `Diagnostics does not have an explanation for what this means. Please report this error so one can be added.`
				}
				log.Errorf("(%T) %v\n%s", err, err, reason)
			} else {
				log.Info("Successfully requested project list from OpenShift master")
			}
		},
	},
}

func TestContext(contextName string, config *clientcmdapi.Config) (result string, success bool) {
	context, exists := config.Contexts[contextName]
	if !exists {
		return "kubeconfig context '" + contextName + "' is not defined.", false
	}
	clusterName := context.Cluster
	cluster, exists := config.Clusters[clusterName]
	if !exists {
		return fmt.Sprintf("kubeconfig context '%s' has a cluster '%s' which is not defined.", contextName, clusterName), false
	}
	project := context.Namespace
	if project == "" {
		project = "default" // OpenShift fills this in
	}
	// TODO: actually send a request to see if can connect
	return fmt.Sprintf(`
The server URL is '%s'
The user authentication is '%s'
The project is '%s'`, cluster.Server, context.AuthInfo, project), true
}
