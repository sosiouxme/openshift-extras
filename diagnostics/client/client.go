package client

import (
	"fmt"
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	//"github.com/kr/pretty"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
)

var Diagnostics = map[string]types.Diagnostic{
	"NodeDefinitions": types.Diagnostic{
		Description: "Check node records on master",
		Condition: func(env *types.Environment) (skip bool, reason string) {
			if env.ClusterAdminFactory == nil {
				return true, "Client does not have cluster-admin access and cannot see node records"
			}
			return false, ""
		},
		Run: func(env *types.Environment) {
			var err error
			var nodes *kapi.NodeList
			if _, kclient, err := env.ClusterAdminFactory.Clients(); err == nil {
				nodes, err = kclient.Nodes().List()
			}
			if err != nil {
				log.Errorf("clGetNodesFailed", `
Client error while retrieving node records. Client retrieved records
during discovery, so this is likely to be a transient error. Try running
diagnostics again. If this message persists, there may be a permissions
problem with getting node records. The error was:

(%T) %[1]v`, err)
				return
			}
			for _, node := range nodes.Items {
				//pretty.Println("Node record:", node)
				var ready, schedulable *kapi.NodeCondition
				for i, condition := range node.Status.Conditions {
					switch condition.Type {
					case kapi.NodeReady:
						ready = &node.Status.Conditions[i]
					case kapi.NodeSchedulable:
						schedulable = &node.Status.Conditions[i]
					}
				}
				//pretty.Println("Node conditions for "+node.Name, ready, schedulable)
				if schedulable != nil && schedulable.Status == kapi.ConditionFull && (ready == nil || ready.Status != kapi.ConditionFull) {
					msg := log.Msg{
						"node": node.Name,
						"tmpl": `
Node {{.node}} is allowed to have pods scheduled but is not ready to
run them. Ready status is {{.status}} because "{{.reason}}"

While in this state, pods could be scheduled to deploy on the node but
will not be deployed until the node achieves readiness.`,
					}
					if ready == nil {
						msg["status"] = "None"
						msg["reason"] = "There is no readiness record."
					} else {
						msg["status"] = ready.Status
						msg["reason"] = ready.Reason
					}
					log.Errorm("clNodeBroken", msg)
				}
			}
		},
	},
	"ConfigContexts": types.Diagnostic{
		Description: "Test that client config contexts have no undefined references",
		Condition: func(env *types.Environment) (skip bool, reason string) {
			if env.ClientConfigRaw == nil {
				return true, "There is no client config file"
			}
			return false, ""
		},
		Run: func(env *types.Environment) {
			cc := env.ClientConfigRaw
			current := cc.CurrentContext
			ccSuccess := false
			var ccResult log.Msg //nil
			for context, _ := range cc.Contexts {
				result, success := TestContext(context, cc)
				msg := log.Msg{"tmpl": "For client config context '{{.context}}':{{.result}}", "context": context, "result": result}
				if context == current {
					ccResult, ccSuccess = msg, success
				} else if success {
					log.Infom("clientCfgSuccess", msg)
				} else {
					log.Warnm("clientCfgWarn", msg)
				}
			}
			if _, exists := cc.Contexts[current]; exists {
				ccResult["tmpl"] = `
The current context from client config is '{{.context}}'
This will be used by default to contact your OpenShift server.
` + ccResult["tmpl"].(string)
				if ccSuccess {
					log.Infom("currentccSuccess", ccResult)
				} else {
					log.Errorm("currentccWarn", ccResult)
				}
			} else { // context does not exist
				log.Errorm("cConUndef", log.Msg{"tmpl": `
Your client config specifies a current context of '{{.context}}'
which is not defined; it is likely that a mistake was introduced while
manually editing your config. If this is a simple typo, you may be
able to fix it manually.
The OpenShift master creates a fresh config when it is started; it may be
useful to use this as a base if available.`, "context": current})
			}
		},
	},
}

func TestContext(contextName string, config *clientcmdapi.Config) (result string, success bool) {
	context, exists := config.Contexts[contextName]
	if !exists {
		return "client config context '" + contextName + "' is not defined.", false
	}
	clusterName := context.Cluster
	cluster, exists := config.Clusters[clusterName]
	if !exists {
		return fmt.Sprintf("client config context '%s' has a cluster '%s' which is not defined.", contextName, clusterName), false
	}
	authName := context.AuthInfo
	if _, exists := config.AuthInfos[authName]; !exists {
		return fmt.Sprintf("client config context '%s' has a user identity '%s' which is not defined.", contextName, authName), false
	}
	project := context.Namespace
	if project == "" {
		project = kapi.NamespaceDefault // OpenShift/k8s fills this in if missing
	}
	// TODO: actually send a request to see if can connect
	return fmt.Sprintf(`
The server URL is '%s'
The user authentication is '%s'
The current project is '%s'`, cluster.Server, authName, project), true
}
