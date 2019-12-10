package importer

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// DefaultCLIOptions are the CLIOptions we use everywhere since many interfaces ask for it but we
// always use the default options.
var DefaultCLIOptions = &genericclioptions.ConfigFlags{}
