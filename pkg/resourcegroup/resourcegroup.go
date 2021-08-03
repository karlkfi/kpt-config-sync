package resourcegroup

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/common"
)

// Unstructured creates a ResourceGroup object
func Unstructured(name, namespace, id string) *unstructured.Unstructured {
	groupVersion := fmt.Sprintf("%s/%s", live.ResourceGroupGVK.Group, live.ResourceGroupGVK.Version)
	inventoryObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": groupVersion,
			"kind":       live.ResourceGroupGVK.Kind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					common.InventoryLabel: id,
				},
			},
			"spec": map[string]interface{}{
				"resources": []interface{}{},
			},
		},
	}
	return inventoryObj
}
