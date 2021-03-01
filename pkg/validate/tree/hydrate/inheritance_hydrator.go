package hydrate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type inheritanceSpecs map[schema.GroupKind]transform.InheritanceSpec

// Inheritance hydrates the given Tree objects by copying inherited objects from
// abstract namespaces down into child namespaces.
func Inheritance(objs *objects.Tree) status.MultiError {
	if objs.Tree == nil {
		return nil
	}
	specs, err := buildInheritanceSpecs(objs.HierarchyConfigs)
	if err != nil {
		return err
	}
	return specs.visitTreeNode(objs.Tree, nil)
}

// buildInheritanceSpecs populates the InheritanceHydrator with InheritanceSpecs
// based upon the HierarchyConfigs in the system directory.
func buildInheritanceSpecs(objs []ast.FileObject) (inheritanceSpecs, status.Error) {
	specs := make(map[schema.GroupKind]transform.InheritanceSpec)
	for _, obj := range objs {
		s, err := obj.Structured()
		if err != nil {
			return nil, err
		}
		hc := s.(*v1.HierarchyConfig)
		for _, r := range hc.Spec.Resources {
			effectiveMode := r.HierarchyMode
			if r.HierarchyMode == v1.HierarchyModeDefault {
				effectiveMode = v1.HierarchyModeInherit
			}

			for _, k := range r.Kinds {
				gk := schema.GroupKind{Group: r.Group, Kind: k}
				specs[gk] = transform.InheritanceSpec{Mode: effectiveMode}
			}
		}
	}
	return specs, nil
}

// visitTreeNode recursively hydrates Namespaces by copying inherited resource
// objects down into child Namespaces.
func (i inheritanceSpecs) visitTreeNode(node *ast.TreeNode, inherited []ast.FileObject) status.MultiError {
	var nodeObjs []ast.FileObject
	isNamespace := false
	for _, o := range node.Objects {
		if o.GroupVersionKind() == kinds.Namespace() {
			isNamespace = true
		} else if o.GroupVersionKind() != kinds.NamespaceSelector() {
			// Don't copy down NamespaceSelectors.
			nodeObjs = append(nodeObjs, o.FileObject)
		}
	}

	if isNamespace {
		return hydrateNamespace(node, inherited)
	}

	err := i.validateAbstractObjects(nodeObjs)
	inherited = append(inherited, nodeObjs...)
	for _, c := range node.Children {
		err = status.Append(err, i.visitTreeNode(c, inherited))
	}
	return err
}

// validateAbstractObjects returns an error if any invalid objects are declared
// in an abstract namespace.
func (i inheritanceSpecs) validateAbstractObjects(objs []ast.FileObject) status.MultiError {
	var err status.MultiError
	for _, o := range objs {
		gvk := o.GroupVersionKind()
		spec, found := i[gvk.GroupKind()]
		if (found && spec.Mode == v1.HierarchyModeNone) && !transform.IsEphemeral(gvk) && !syntax.IsSystemOnly(gvk) {
			err = status.Append(err, validation.IllegalAbstractNamespaceObjectKindError(o))
		}
	}
	return err
}

func hydrateNamespace(node *ast.TreeNode, inherited []ast.FileObject) status.MultiError {
	var err status.MultiError
	for _, child := range node.Children {
		err = status.Append(err, validation.IllegalNamespaceSubdirectoryError(child, node))
	}
	for _, obj := range inherited {
		node.Objects = append(node.Objects, &ast.NamespaceObject{FileObject: obj.DeepCopy()})
	}
	return err
}

// TODO(b/178219594): Move IllegalAbstractNamespaceObjectKindError and  IllegalNamespaceSubdirectoryError here.
