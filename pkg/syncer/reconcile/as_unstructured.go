package reconcile

import (
	"encoding/json"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AsUnstructured attempts to convert a client.Object to an
// *unstructured.Unstructured.
// TODO(b/162260725): This adds .status and .metadata.creationTimestamp to
//  everything. Evaluate every use, and convert to using AsUnstructuredSanitized
//  if possible.
func AsUnstructured(o core.Object) (*unstructured.Unstructured, status.Error) {
	if u, isUnstructured := o.(*unstructured.Unstructured); isUnstructured {
		// The path below returns a deep copy, so we want to make sure we return a
		// deep copy here as well (for symmetry and to avoid subtle bugs).
		return u.DeepCopy(), nil
	}

	jsn, err := json.Marshal(o)
	if err != nil {
		return nil, status.InternalErrorBuilder.Wrap(err).BuildWithResources(o)
	}

	u := &unstructured.Unstructured{}
	err = u.UnmarshalJSON(jsn)
	if err != nil {
		return nil, status.InternalErrorBuilder.Wrap(err).BuildWithResources(o)
	}
	return u, nil
}

// AsUnstructuredSanitized converts o to an Unstructured and removes problematic
// fields:
// - metadata.creationTimestamp
// - status
//
// There is no other way to do this without defining our own versions of the
// Kubernetes type definitions.
// Explanation of why: https://www.sohamkamani.com/golang/2018-07-19-golang-omitempty/
func AsUnstructuredSanitized(o core.Object) (*unstructured.Unstructured, status.Error) {
	u, err := AsUnstructured(o)
	if err != nil {
		return nil, err
	}

	unstructured.RemoveNestedField(u.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(u.Object, "status")
	return u, nil
}
