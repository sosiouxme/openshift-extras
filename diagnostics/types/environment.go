package types

import (
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	osclient "github.com/openshift/origin/pkg/cmd/util/clientcmd"
)

type Environment struct {
	OS         string // "linux / windows / darwin" http://golang.org/pkg/runtime/#GOOS
	HasSystemd bool
	HasBash    bool

	OscPath          string
	OscVersion       Version
	OpenshiftPath    string
	OpenshiftVersion Version
	//KubeconfigPath   string
	OsConfig   *osclient.Config
	Kubeconfig *clientcmdapi.Config // for analysis, not use

	Flags *flags.Flags // user flags deposit results here
}
