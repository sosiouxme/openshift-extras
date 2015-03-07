package types

import (
	"github.com/openshift/openshift-extras/diagnostics/cmd/flags"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
)

type Environment struct {
	OS         string // "linux / windows / darwin" http://golang.org/pkg/runtime/#GOOS
	HasSystemd bool
	HasBash    bool

	OscPath          string
	OscVersion       Version
	OpenshiftPath    string
	OpenshiftVersion Version
	KubeconfigPath   string
	Kubeconfig       *clientcmdapi.Config

	Flags *flags.Flags
}
