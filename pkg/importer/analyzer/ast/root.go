package ast

import (
	"time"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
)

// Root represents a hierarchy of declared configs, settings for how those configs will be
// interpreted, and information regarding where those configs came from.
type Root struct {
	// ImportToken is the token for context
	ImportToken string
	LoadTime    time.Time // Time at which the context was generated

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
	Data *Extension
}

// Accept invokes VisitRoot on the visitor.
func (c *Root) Accept(visitor Visitor) *Root {
	if c == nil {
		return nil
	}
	return visitor.VisitRoot(c)
}
