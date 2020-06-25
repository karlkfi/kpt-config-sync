package bugreport

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/policycontroller"
)

// Product describes an ACM Product
type Product string

const (
	// PolicyController policy controller
	PolicyController = Product("Policy Controller")
	// ConfigSync config sync, AKA Nomos, AKA original ACM
	ConfigSync = Product("Config Sync")
	// KCC AKA CNRM
	KCC = Product("KCC")
)

var (
	productNamespaces = map[Product]string{
		PolicyController: policycontroller.NamespaceSystem,
		KCC:              "cnrm-system",
		ConfigSync:       configmanagement.ControllerNamespace,
	}
)
