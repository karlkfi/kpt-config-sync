package fake

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ObjectList is a collection of runtime.objects.
type ObjectList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Roles
	Items []runtime.Object `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// DeepCopyObject implements runtime.Object.
func (l ObjectList) DeepCopyObject() runtime.Object {
	result := ObjectList{}
	result.TypeMeta = l.TypeMeta
	l.ListMeta.DeepCopyInto(&result.ListMeta)

	result.Items = make([]runtime.Object, len(l.Items))
	for i, obj := range l.Items {
		result.Items[i] = obj.DeepCopyObject()
	}
	return &result
}
