package fake

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	v1 "k8s.io/api/core/v1"
)

// PodObject returns an initialized Pod.
func PodObject(name string, containers []v1.Container, opts ...core.MetaMutator) *v1.Pod {
	result := &v1.Pod{
		TypeMeta: ToTypeMeta(kinds.Pod()),
		Spec:     v1.PodSpec{Containers: containers},
	}
	mutate(result, core.Name(name))
	mutate(result, opts...)

	return result
}
