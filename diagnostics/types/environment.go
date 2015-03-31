package types

import (
	kubeclient "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kclientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	osclient "github.com/openshift/origin/pkg/client"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/spf13/cobra"
)

type Environment struct {
	OS           string // "linux / windows / darwin" http://golang.org/pkg/runtime/#GOOS
	HasSystemd   bool
	HasBash      bool
	SystemdUnits map[string]SystemdUnit // list of those present on system

	OscPath             string
	OscVersion          Version
	OpenshiftPath       string
	OpenshiftVersion    Version
	ClientConfigPath    string                          // first client config file found, if any
	ClientConfigRaw     *kclientcmdapi.Config           // available to analyze ^^
	OsConfig            *kclientcmdapi.Config           // actual merged client configuration
	FactoryForContext   map[string]*osclientcmd.Factory // one for each known context
	AccessForContext    map[string]*ContextAccess       // for each context that has access to anything
	ClusterAdminFactory *osclientcmd.Factory            // factory we will use for cluster-admin access (could easily be nil)
	// TODO: These should go away ASAP, in favor of using ^^:
	Factory    *osclientcmd.Factory // for generating client from configs
	OsClient   *osclient.Client
	KubeClient *kubeclient.Client
	Command    *cobra.Command

	Flags *flags.Flags // command flags deposit results here; also has command flag objects
}

type ContextAccess struct {
	Projects     []string
	ClusterAdmin bool // has access to see stuff only cluster-admin should
}

func NewEnvironment(fl *flags.Flags, f *osclientcmd.Factory, c *cobra.Command) *Environment {
	return &Environment{
		Flags:             fl,
		Factory:           f,
		Command:           c,
		SystemdUnits:      make(map[string]SystemdUnit),
		FactoryForContext: make(map[string]*osclientcmd.Factory),
		AccessForContext:  make(map[string]*ContextAccess),
	}
}

// helpful translator
func (env *Environment) DefaultFactory() *osclientcmd.Factory {
	if env.FactoryForContext != nil && env.OsConfig != nil { // no need to panic if missing...
		return env.FactoryForContext[env.OsConfig.CurrentContext]
	}
	return nil
}
