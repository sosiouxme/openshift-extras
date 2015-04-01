package client

import (
	"fmt"
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
)

var Diagnostics = map[string]types.Diagnostic{
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
