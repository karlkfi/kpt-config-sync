package operators

// NOTE: these types have been taken over from cluster-operators source code.
// Please do not regenerate.
//
// Comments have been added to satisfy the nomos linter which is more nitpicky
// than cluster-operators one.

import (
	addonsv1alpha1 "github.com/google/nomos/pkg/installer/cluster-operators/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StandardOperatorObject is an interface.
type StandardOperatorObject interface {
	runtime.Object
	metav1.Object
	CommonSpec() CommonSpec
	GetCommonStatus() CommonStatus
	SetCommonStatus(CommonStatus)
}

// CommonSpec is a struct.
type CommonSpec struct {
	Version string `json:"version,omitempty"`
	Channel string `json:"channel,omitempty"`
}

// CommonStatus is a struct.
type CommonStatus struct {
	Healthy     bool                              `json:"healthy,omitempty"`
	Services    []addonsv1alpha1.ServiceStatus    `json:"services,omitempty"`
	Deployments []addonsv1alpha1.DeploymentStatus `json:"deployments,omitempty"`
}
