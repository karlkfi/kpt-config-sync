package testoutput

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
)

// Source annotates the object as being declared in a specific source file.
func Source(path string) core.MetaMutator {
	return core.Annotation(v1.SourcePathAnnotationKey, path)
}

// InCluster annotates the object as being in a specific cluster.
func InCluster(clusterName string) core.MetaMutator {
	return core.Annotation(v1.ClusterNameAnnotationKey, clusterName)
}
