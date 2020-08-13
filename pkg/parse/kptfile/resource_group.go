package kptfile

import (
	"github.com/google/nomos/pkg/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	group       = "configmanagement.gke.io"
	version     = "v1beta1"
	kind        = "ResourceGroup"
	application = "application"
)

// ResourceGroupSpec defines the spec of ResourceGroup.
type ResourceGroupSpec struct {
	Resources  []ObjMetadata `json:"resources,omitempty"`
	Descriptor Descriptor    `json:"descriptor,omitempty"`
}

// ObjMetadata organizes and stores the identifying information
// for an object.
type ObjMetadata struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Group     string `json:"group"`
	Kind      string `json:"kind"`
}

// Descriptor defines the type of ResourceGroup.
type Descriptor struct {
	Type string `json:"type,omitempty"`
}

// ResourceGroup defines the schema of ResourceGroup.
// The actual type is defined in
//  https://gke-internal.googlesource.com/GoogleCloudPlatform/resource-group/+/f8e2a2f575e5d8b9bf336a2cfadc0c95ea98db9e/api/v1beta1/resourcegroup_types.go.
// Here only spec is defined and status is ignored.
// Here is an example of ResourceGroup:
//    apiVersion: configmanagement.gke.io/v1beta1
//    kind: ResourceGroup
//    metadata:
//      name: resourcegroup-sample
//      namespace: default
//    spec:
//      resources:
//      - name: foo
//        namespace: default
//        group: ""
//        kind: ConfigMap
//      - name: helloworld-gke
//        namespace: newapp-prod2
//        group: apps
//        kind: Deployment
//      - name: helloworld-gke
//        namespace: otherformat-prod3
//        group: apps
//        kind: Deployment
//      - group: apiextensions.k8s.io
//        kind: CustomResourceDefinition
//        name: crontabs.stable.example.com
//        namespace: ""
type ResourceGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceGroupSpec `json:"spec,omitempty"`
}

func (in *ResourceGroup) deepCopy() *ResourceGroup {
	if in == nil {
		return nil
	}
	out := new(ResourceGroup)
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec.Descriptor = in.Spec.Descriptor
	out.Spec.Resources = in.Spec.Resources
	return out
}

// DeepCopyObject copies the receiver and creates a new runtime.Object.
func (in *ResourceGroup) DeepCopyObject() runtime.Object {
	if c := in.deepCopy(); c != nil {
		return c
	}
	return nil
}

// resourceGroupFromKptFile creates a ResourceGroup based on the inventory field
// of a KptFile with the following mapping:
// inventory.identifer -> resourcegroup name
// inventory.namespace -> resourcegroup namespace
// inventory.annotations -> resourcegroup annotations
// inventory.labels -> resourcegroup labels
func resourceGroupFromKptFile(kpt *Kptfile, ids []ObjMetadata) core.Object {
	name := kpt.Inventory.Identifier
	namespace := kpt.Inventory.Namespace
	labels := kpt.Inventory.Labels
	annotations := kpt.Inventory.Annotations

	rg := newResourceGroup(name, namespace, labels, annotations, ids)
	return rg
}

func newResourceGroup(name, namespace string, labels, annotations map[string]string, ids []ObjMetadata) *ResourceGroup {
	rg := &ResourceGroup{}
	rg.SetName(name)
	rg.SetNamespace(namespace)
	rg.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	})
	rg.SetLabels(deepCopyMap(labels))
	rg.SetAnnotations(deepCopyMap(annotations))
	rg.Spec.Descriptor = Descriptor{Type: application}
	rg.Spec.Resources = make([]ObjMetadata, len(ids))
	copy(rg.Spec.Resources, ids)

	return rg
}
