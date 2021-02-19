package bugreport

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/policycontroller"
)

// Product describes an ACM Product
type Product string

const (
	// PolicyController policy controller
	PolicyController = Product("Policy Controller")
	// ConfigSync config sync, AKA Nomos, AKA original ACM
	ConfigSync = Product("Config Sync")
	// ConfigSyncMonitoring controller
	ConfigSyncMonitoring = Product("Config Sync Monitoring")
	// KCC AKA CNRM
	KCC = Product("KCC")
	// ResourceGroup controller
	ResourceGroup = Product("Resource Group")
)

// Resource Group constants
const (
	// RGControllerNamespace is the namespace used for the resource-group controller
	RGControllerNamespace = "resource-group-system"
)

var (
	productNamespaces = map[Product]string{
		PolicyController:     policycontroller.NamespaceSystem,
		KCC:                  "cnrm-system",
		ConfigSync:           configmanagement.ControllerNamespace,
		ResourceGroup:        RGControllerNamespace,
		ConfigSyncMonitoring: metrics.MonitoringNamespace,
	}
)
