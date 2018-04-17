package syncer

import (
	"github.com/google/nomos/pkg/client/action"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Module is a type specific implementation for resource synchronization.
// Each module is responsible for synchronizing only one resource type.
type Module interface {
	// Name is the name of the module, this will mainly be used for logging
	// purposes.
	Name() string

	// Equal must implement a function that determines if the content
	// fields of the two objects are equivalent. This should only consider
	// annotations if they are overloaded to hold additional data for the
	// object.
	Equal(meta_v1.Object, meta_v1.Object) bool

	// InformerProvider returns an informer provider for the controlled
	// resource type.
	InformerProvider() informers.InformerProvider

	// Instance returns an instance of the type that this module is going
	// to be synchronizing. Since it operates on API types, they should all
	// satisfy the meta_v1.Object interface.
	Instance() meta_v1.Object

	// ActionSpec returns the spec for the API type that this module will
	// be synchronizing. This should correspond to a spec for the same type
	// that Instance returns.
	ActionSpec() *action.ReflectiveActionSpec
}
