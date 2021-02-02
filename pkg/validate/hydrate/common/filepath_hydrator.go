package common

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
)

type filepathHydrator struct {
	policyDir cmpath.Relative
}

var _ parsed.Hydrator = &filepathHydrator{}

// FilepathFlatHydrator returns a FlatHydrator that annotates all cluster-scoped
// namespace-scoped objects with the filepath where they are declared.
func FilepathFlatHydrator(policyDir cmpath.Relative) parsed.FlatHydrator {
	return parsed.FlatWrap(&filepathHydrator{policyDir})
}

// FilepathTreeHydrator returns a TreeHydrator that annotates all cluster-scoped
// namespace-scoped objects with the filepath where they are declared.
func FilepathTreeHydrator(policyDir cmpath.Relative) parsed.TreeHydrator {
	return parsed.TreeWrap(&filepathHydrator{policyDir})
}

// Hydrate implements parsed.Hydrator
func (f *filepathHydrator) Hydrate(root parsed.Root) status.MultiError {
	return status.Append(
		root.VisitClusterObjects(parsed.PerObjectVisitor(f.addFilepathAnnotation)),
		root.VisitNamespaceObjects(parsed.PerObjectVisitor(f.addFilepathAnnotation)),
	)
}

func (f *filepathHydrator) addFilepathAnnotation(obj ast.FileObject) status.Error {
	core.SetAnnotation(obj, v1.SourcePathAnnotationKey, f.policyDir.Join(obj.Relative).SlashPath())
	return nil
}
