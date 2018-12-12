package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/runtime"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ClusterregistryKindValidator ensures only the allowed set of types appear in clusterregistry/
var ClusterregistryKindValidator = &ObjectValidator{
	validate: func(source string, object runtime.Object) error {
		switch o := object.(type) {
		case *v1alpha1.ClusterSelector:
		case *clusterregistry.Cluster:
		default:
			return vet.IllegalKindInClusterregistryError{Source: source, GroupVersionKind: o.GetObjectKind().GroupVersionKind()}
		}
		return nil
	},
}
