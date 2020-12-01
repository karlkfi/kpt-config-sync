package resourcegroup

import (
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	version     = "v1alpha1"
	kind        = "ResourceGroup"
	application = "Application"
)

// Spec defines the spec of ResourceGroup.
type Spec struct {
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
//  https://gke-internal.googlesource.com/GoogleCloudPlatform/resource-group/+/refs/heads/master/api/v1alpha1/resourcegroup_types.go
// Here only spec is defined and status is ignored.
// Here is an example of ResourceGroup:
//    apiVersion: configsync.gke.io/v1alpha1
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
	Spec              Spec `json:"spec,omitempty"`
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

// FromKptFile creates a ResourceGroup based on the inventory field
// of a KptFile with the following mapping:
// inventory.name -> resourcegroup name
// inventory.namespace -> resourcegroup namespace
// inventory.annotations -> resourcegroup annotations
// inventory.labels -> resourcegroup labels
func FromKptFile(kf *kptfile.KptFile, ids []ObjMetadata) core.Object {
	name := kf.Inventory.Name
	namespace := kf.Inventory.Namespace
	labels := kf.Inventory.Labels
	annotations := kf.Inventory.Annotations

	rg := NewResourceGroup(name, namespace, labels, annotations, ids)
	return rg
}

// NewResourceGroup constructs a *ResourceGroup from the passed settings.
func NewResourceGroup(name, namespace string, labels, annotations map[string]string, ids []ObjMetadata) *ResourceGroup {
	rg := &ResourceGroup{}
	rg.SetName(name)
	rg.SetNamespace(namespace)
	rg.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   configsync.GroupName,
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

func deepCopyMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
