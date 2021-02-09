package fake

import (
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/google/nomos/pkg/core"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResourceGroupObject initializes a ResourceGroup.
func ResourceGroupObject(opts ...core.MetaMutator) *unstructured.Unstructured {
	result := live.ResourceGroupUnstructured("", "", "")
	defaultMutate(result)
	mutate(result, opts...)

	return result
}
