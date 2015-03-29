package discovery // config

import (
	"fmt"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	"github.com/openshift/openshift-extras/diagnostics/log"
	"github.com/openshift/openshift-extras/diagnostics/types"
	"io/ioutil"
	"os"
)

/* ----------------------------------------------------------
Look for the .kubeconfig and read it in.

We are not going to attempt merging multiple files as the
actual client permits; most users will just be dealing with
a single kubeconfig, and we will be replacing all of that
with OpenShift-specific config soon enough.

We will look in the standard locations, alert the user to
what we find as we go along, and try to be helpful.
*/
func readKubeconfig(env *types.Environment) {
	file := findKubeconfig(env)
	if file == nil {
		log.Warn("discNoKC", "No .kubeconfig read; default OpenShift config will be used, which is likely not what you want.")
	} else {
		defer file.Close()
		if buffer, err := ioutil.ReadAll(file); err != nil {
			log.Errorf("discKCReadErr", "Unexpected error while reading .kubeconfig file (%s): %v", file.Name(), err)
		} else {
			if config, err := clientcmd.Load(buffer); err != nil {
				log.Errorf("discKCYamlErr", `Error reading YAML from kubeconfig file (%s):
  %v
This file may have been truncated or mis-edited.
Please fix or get a new .kubeconfig`, file.Name(), err)
			} else {
				/* Note, we're not actually going to use this config,
				 * because it's too hard to turn it into something that's useful.
				 * Instead, we'll defer to the openshift client code to assimilate
				 * flags, env vars, and the potential hierarchy of kubeconfig files
				 * into an actual configuration that the client uses.
				 */
				env.Kubeconfig = config
				log.Infom("discKCRead", log.Msg{"tmpl": `Successfully read a .kubeconfig file at '{{.path}}';
be aware that the actual configuration used later may be different
due to environment variables, flags, and other .kubeconfig files
being merged together.`, "path": file.Name()})
			}
		}
	}
}

// ----------------------------------------------------------
// Look for .kubeconfig in a number of possible locations
func findKubeconfig(env *types.Environment) (file *os.File) {
	fPath := env.Flags.OpenshiftFlags.Lookup("config").Value.String()
	kcPath := os.Getenv("KUBECONFIG")
	adminPath1 := "/var/lib/openshift/openshift.certificates.d/admin/.kubeconfig" // enterprise
	adminPath2 := "/openshift.certificates.d/admin/.kubeconfig"                   // origin systemd
	adminWarningF := `
.kubeconfig was not available where expected; however, one exists at
  %s
which is a standard location where the master generates it.
If this is what you want, you should copy it to a standard location
(your home directory, or the current directory), or you can set the
environment variable KUBECONFIG in your ~/.bash_profile:
  export KUBECONFIG=%s
If this is not what you want, you should obtain a .kubeconfig and
place it in a standard location.
`
	if fPath != "" {
		// user specified intended path; don't keep looking if it isn't there.
		return openKubeconfig(fPath, fmt.Sprintf("-c specified that .kubeconfig should be at %s\n", fPath))
	} else if kcPath != "" {
		// $KUBECONFIG specified intended path; don't keep looking if it isn't there
		return openKubeconfig(kcPath, fmt.Sprintf("$KUBECONFIG specified that .kubeconfig should be at %s\n", kcPath))
	}
	// look for it in `pwd`
	path, _ := os.Getwd()
	if file = openKubeconfig(path+"/.kubeconfig", ""); file != nil {
		return file
	}
	// look for it in $HOME
	if file = openKubeconfig(os.Getenv("HOME")+"/.kube/.kubeconfig", ""); file != nil {
		return file
	}
	// look for it in auto-generated locations when not found properly
	if file = openKubeconfig(adminPath1, ""); file != nil {
		file.Close()
		file = nil
		log.Warnf("discKCautoPath", adminWarningF, adminPath1, adminPath1)
	}
	if file = openKubeconfig(adminPath2, ""); file != nil {
		file.Close()
		file = nil
		log.Warnf("discKCautoPath", adminWarningF, adminPath2, adminPath2)
	}
	return file
}

// ----------------------------------------------------------
// Attempt to open file at path as .kubeconfig
// If there is a problem and errmsg is set, log an error
func openKubeconfig(path string, errmsg string) (file *os.File) {
	var err error
	if path != "" {
		if file, err = os.Open(path); err == nil {
			log.Infom("discOpenKC", log.Msg{"tmpl": "Reading .kubeconfig at {{.path}}", "path": path})
		} else if errmsg == "" {
			log.Debugf("discOpenKCNo", "Could not read .kubeconfig at %s:\n%#v", path, err)
		} else if os.IsNotExist(err) {
			log.Error("discOpenKCNoExist", errmsg+"but that file does not exist.")
		} else if os.IsPermission(err) {
			log.Error("discOpenKCNoPerm", errmsg+"but lack permission to read that file.")
		} else if err != nil {
			log.Errorf("discOpenKCErr", "%sbut there was an error opening it:\n%#v", errmsg, err)
		} // else it is open for reading
	}
	return file
}
