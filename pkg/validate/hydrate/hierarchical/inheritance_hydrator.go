package hierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// inheritanceHydrator is a TreeHydrator that copies inherited objects from
// abstract namespaces down into child Namespaces.
type inheritanceHydrator struct {
	specs map[schema.GroupKind]transform.InheritanceSpec
}

var _ parsed.TreeHydrator = &inheritanceHydrator{}

// InheritanceHydrator returns an instantiated InheritanceHydrator.
func InheritanceHydrator() parsed.TreeHydrator {
	return &inheritanceHydrator{
		specs: map[schema.GroupKind]transform.InheritanceSpec{},
	}
}

// Hydrate implements TreeHydrator.
func (h *inheritanceHydrator) Hydrate(root *parsed.TreeRoot) status.MultiError {
	if err := root.VisitSystemObjects(h.buildInheritanceSpecs); err != nil {
		return err
	}
	if root.Tree == nil {
		return nil
	}
	return h.visitTreeNode(root.Tree, nil)
}

// buildInheritanceSpecs populates the InheritanceHydrator with InheritanceSpecs
// based upon the HierarchyConfigs in the system directory.
func (h *inheritanceHydrator) buildInheritanceSpecs(objs []ast.FileObject) status.MultiError {
	for _, obj := range objs {
		if obj.GroupVersionKind() != kinds.HierarchyConfig() {
			continue
		}
		s, err := obj.Structured()
		if err != nil {
			return err
		}
		hc := s.(*v1.HierarchyConfig)
		for _, r := range hc.Spec.Resources {
			effectiveMode := r.HierarchyMode
			if r.HierarchyMode == v1.HierarchyModeDefault {
				effectiveMode = v1.HierarchyModeInherit
			}

			for _, k := range r.Kinds {
				gk := schema.GroupKind{Group: r.Group, Kind: k}
				h.specs[gk] = transform.InheritanceSpec{Mode: effectiveMode}
			}
		}
	}
	return nil
}

// visitTreeNode recursively hydrates Namespaces by copying inherited resource
// objects down into child Namespaces.
func (h *inheritanceHydrator) visitTreeNode(node *ast.TreeNode, inherited []ast.FileObject) status.MultiError {
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

	err := h.validateAbstractObjects(nodeObjs)
	inherited = append(inherited, nodeObjs...)
	for _, c := range node.Children {
		err = status.Append(err, h.visitTreeNode(c, inherited))
	}
	return err
}

// validateAbstractObjects returns an error if any invalid objects are declared
// in an abstract namespace.
func (h *inheritanceHydrator) validateAbstractObjects(objs []ast.FileObject) status.MultiError {
	var err status.MultiError
	for _, o := range objs {
		gvk := o.GroupVersionKind()
		spec, found := h.specs[gvk.GroupKind()]
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
