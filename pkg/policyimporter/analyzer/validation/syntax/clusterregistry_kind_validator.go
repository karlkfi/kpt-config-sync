package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ClusterregistryKindValidator ensures only the allowed set of types appear in clusterregistry/
var ClusterregistryKindValidator = &FileObjectValidator{
	validate: func(object ast.FileObject) error {
		switch o := object.Object.(type) {
		case *v1alpha1.ClusterSelector:
		case *clusterregistry.Cluster:
		default:
			return vet.IllegalKindInClusterregistryError{Source: object.Source, GroupVersionKind: o.GetObjectKind().GroupVersionKind()}
		}
		return nil
	},
}
