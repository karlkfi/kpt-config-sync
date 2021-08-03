package fake

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/resourcegroup"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResourceGroupObject initializes a ResourceGroup.
func ResourceGroupObject(opts ...core.MetaMutator) *unstructured.Unstructured {
	result := resourcegroup.Unstructured("", "", "")
	defaultMutate(result)
	mutate(result, opts...)

	return result
}
