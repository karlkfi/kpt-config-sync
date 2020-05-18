package ast

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
)

// Root represents a hierarchy of declared configs, settings for how those configs will be
// interpreted, and information regarding where those configs came from.
type Root struct {
	// ClusterName is the name of the Cluster to generate the policy hierarchy for. Determines which
	// ClusterSelectors are active.
	ClusterName string
	Repo        *v1.Repo // Nomos repo

	// ClusterObjects represents resources that are cluster scoped.
	ClusterObjects []*ClusterObject

	// ClusterRegistryObjects represents resources that are related to multi-cluster.
	ClusterRegistryObjects []*ClusterRegistryObject

	// SystemObjects represents resources regarding nomos configuration.
	SystemObjects []*SystemObject

	// Tree represents the directory hierarchy containing namespace scoped resources.
	Tree *TreeNode
}

// Accept invokes VisitRoot on the visitor.
func (r *Root) Accept(visitor Visitor) *Root {
	if r == nil {
		return nil
	}
	return visitor.VisitRoot(r)
}

// Flatten returns a list of all materialized objects in the Root.
// It returns all objects that we would actually sync to a cluster.
func (r *Root) Flatten() []FileObject {
	if r == nil {
		return nil
	}
	var result []FileObject

	for _, o := range r.SystemObjects {
		result = append(result, o.FileObject)
	}

	for _, o := range r.ClusterObjects {
		result = append(result, o.FileObject)
	}

	for _, o := range r.ClusterRegistryObjects {
		result = append(result, o.FileObject)
	}

	if r.Tree != nil {
		result = append(result, r.Tree.flatten()...)
	}

	return result
}
