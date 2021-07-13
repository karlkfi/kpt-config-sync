package core

import (
	"encoding/json"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func init() {
	// Add ConfigSync types to the Scheme used by RemarshalToStructured for
	// converting Unstructured to specific types.
	utilruntime.Must(apiextensionsv1beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(corev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterregistry.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme.Scheme))
}

// RemarshalToStructured converts a runtime.Object to the literal Go struct, if
// one is available. Returns an error if this process fails.
func RemarshalToStructured(u *unstructured.Unstructured) (runtime.Object, error) {
	result, err := scheme.Scheme.New(u.GetObjectKind().GroupVersionKind())
	if err != nil {
		return nil, err
	}

	jsn, err := u.MarshalJSON()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsn, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
