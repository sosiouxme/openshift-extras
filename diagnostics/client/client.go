package client

import (
	"fmt"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
)

var diagnostics = map[string]func(env *types.Environment){
	// Test that clusters and current cluster in kubeconfig are sane
	"CheckClusters": func(env *types.Environment) {
		log.Info("Testing server configuration(s) from kubeconfig")
		kc := env.Kubeconfig
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
}

func Diagnose(env *types.Environment) {
	for name, f := range diagnostics {
		log.Infof("\n### Running diagnostic: client.%s ###\n", name)
		f(env)
	}
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
	return fmt.Sprintf(`
The server URL is '%s'
The user authentication is '%s'
The project is '%s'`, cluster.Server, context.AuthInfo, project), true
}
