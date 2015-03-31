package client

import (
	"fmt"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
)

var Diagnostics = map[string]types.Diagnostic{
	"KubeconfigContexts": types.Diagnostic{
		Description: "Test that kubeconfig contexts and current context are ok",
		Condition: func(env *types.Environment) (skip bool, reason string) {
			if env.ClientConfigRaw == nil {
				return true, "There is no .kubeconfig file"
			}
			return false, ""
		},
		Run: func(env *types.Environment) {
			kc := env.ClientConfigRaw
			log.Info("kubeconfigTest", "Testing server configuration(s) from kubeconfig")
			cc := kc.CurrentContext
			ccSuccess := false
			var ccResult log.Msg //nil
			for context, _ := range kc.Contexts {
				result, success := TestContext(context, kc)
				msg := log.Msg{"tmpl": "For kubeconfig context '{{.context}}':{{.result}}", "context": context, "result": result}
				if context == cc {
					ccResult, ccSuccess = msg, success
				} else if success {
					log.Infom("kubeconfigSuccess", msg)
				} else {
					log.Warnm("kubeconfigWarn", msg)
				}
				// TODO: actually test whether these contexts are usable
			}
			if _, exists := kc.Contexts[cc]; exists {
				ccResult["tmpl"] = `
The current context from kubeconfig is '{{.context}}'
This will be used by default to contact your OpenShift server.
` + ccResult["tmpl"].(string)
				if ccSuccess {
					log.Infom("currentkcSuccess", ccResult)
				} else {
					log.Errorm("currentkcWarn", ccResult)
				}
			} else { // context does not exist
				log.Errorm("cConUndef", log.Msg{"tmpl": `
Your kubeconfig specifies a current context of '{{.context}}'
which is not defined; it is likely that a mistake was introduced while
manually editing your kubeconfig. If this is a simple typo, you may be
able to fix it manually.
The OpenShift master creates a fresh kubeconfig when it is started; it may be
useful to use this as a base if available.`, "context": cc})
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
