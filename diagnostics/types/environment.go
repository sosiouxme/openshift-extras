package types

import (
	kubeclient "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kccmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
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

	OscPath          string
	OscVersion       Version
	OpenshiftPath    string
	OpenshiftVersion Version
	OsConfig         *osclientcmd.Config
	Kubeconfig       *kccmdapi.Config // for analysis, not configuration
	OsClient         *osclient.Client
	KubeClient       *kubeclient.Client
	Factory          *osclientcmd.Factory
	Command          *cobra.Command

	Flags *flags.Flags // command flags deposit results here
}
