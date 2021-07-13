package testoutput

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
)

// Source annotates the object as being declared in a specific source file.
func Source(path string) core.MetaMutator {
	return core.Annotation(metadata.SourcePathAnnotationKey, path)
}

// InCluster annotates the object as being in a specific cluster.
func InCluster(clusterName string) core.MetaMutator {
	return core.Annotation(metadata.ClusterNameAnnotationKey, clusterName)
}
