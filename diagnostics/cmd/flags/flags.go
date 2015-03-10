package flags

import (
	flag "github.com/spf13/pflag"
)

type Flags struct {
	LogLevel       int
	OpenshiftPath  string
	OscPath        string
	OpenshiftFlags *flag.FlagSet
	//	KubeconfigPath string
}
