package testoutput

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/object"
)

// Source annotates the object as being declared in a specific source file.
func Source(path string) object.MetaMutator {
	return object.Annotation(v1.SourcePathAnnotationKey, path)
}

// InCluster annotates the object as being in a specific cluster.
func InCluster(clusterName string) object.MetaMutator {
	return object.Annotation(v1.ClusterNameAnnotationKey, clusterName)
}
