package fake

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	appsv1 "k8s.io/api/apps/v1"
)

// DeploymentObject initializes a Deployment.
func DeploymentObject(opts ...object.MetaMutator) *appsv1.Deployment {
	result := &appsv1.Deployment{TypeMeta: toTypeMeta(kinds.Deployment())}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// Deployment returns a Deployment in a FileObject.
func Deployment(dir string, opts ...object.MetaMutator) ast.FileObject {
	relative := cmpath.FromSlash(dir).Join("deployment.yaml")
	return FileObject(DeploymentObject(opts...), relative.SlashPath())
}
