package gatekeeper

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// TemplatesGroup is the api group for gatekeeper constraint templates
	TemplatesGroup = "templates.gatekeeper.sh"
	// ConstraintsGroup is the api group for gatekeeper constraints
	ConstraintsGroup = "constraints.gatekeeper.sh"
)

var (
	// ConstraintTemplateGroupKind is the GroupKind for gatekeeper constraint templates
	ConstraintTemplateGroupKind = schema.GroupKind{
		Group: TemplatesGroup,
		Kind:  "ConstraintTemplate",
	}
	// ConstraintTemplateGKV1alpha1 is the v1alpha1 GVK
	ConstraintTemplateGKV1alpha1 = ConstraintTemplateGroupKind.WithVersion("v1alpha1")
	// ConstraintTemplateGKV1beta1 is the v1beta1 GVK
	ConstraintTemplateGKV1beta1 = ConstraintTemplateGroupKind.WithVersion("v1beta1")
)

// ConstraintTemplateCRD converts a gatekeeper constraint template to the
// CRD that gatekeeper will apply to the API server.
// Note that this should eventually attempt to use the gatekeeper libraries, however
// that's going to be a lot of work given golang vendoring and the common k8s
// dependencies.
func ConstraintTemplateCRD(o core.Object) (*v1beta1.CustomResourceDefinition, error) {
	obj, ok := o.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.Errorf("expected unstructured.Unstructured, got %v", o)
	}

	crdNames, found, err := unstructured.NestedMap(obj.Object, "spec", "crd", "spec", "names")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spec from constraint template")
	}
	if !found {
		return nil, errors.Errorf("crd spec not found in constraint template %s", obj.Object)
	}

	kind, _, err := unstructured.NestedString(crdNames, "kind")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get constraint template field")
	}

	// The fields ListKind, plural and singular are not actually required.
	// This reproduces what Gatekeeper is doing.
	listKind := kind + "List"
	plural := strings.ToLower(kind)
	singular := strings.ToLower(kind)

	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", plural, ConstraintsGroup),
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: ConstraintsGroup,
			Names: v1beta1.CustomResourceDefinitionNames{

				Plural:   plural,
				Singular: singular,
				Kind:     kind,
				ListKind: listKind,
			},
			Scope: v1beta1.ClusterScoped,
			Versions: []v1beta1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: false,
				},
				{
					Name:    "v1beta1",
					Served:  true,
					Storage: true,
				},
			},
		},
	}
	crd.SetGroupVersionKind(kinds.CustomResourceDefinition())
	return crd, nil
}
