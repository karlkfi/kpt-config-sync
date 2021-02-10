package common

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func crds(gvks ...schema.GroupVersionKind) []*v1beta1.CustomResourceDefinition {
	var result []*v1beta1.CustomResourceDefinition
	for _, gvk := range gvks {
		result = append(result, &v1beta1.CustomResourceDefinition{
			Spec: v1beta1.CustomResourceDefinitionSpec{
				Group: gvk.Group,
				Names: v1beta1.CustomResourceDefinitionNames{
					Kind: gvk.Kind,
				},
			},
		})
	}
	return result
}

func TestCRDRemovalValidator(t *testing.T) {
	testCases := []struct {
		name     string
		root     parsed.Root
		previous []*v1beta1.CustomResourceDefinition
		current  []*v1beta1.CustomResourceDefinition
		wantErr  status.MultiError
	}{
		{
			name: "no previous or current CRDs",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
		},
		{
			name: "add a CRD",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
			current: crds(kinds.Anvil()),
		},
		{
			name: "keep a CRD",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
			previous: crds(kinds.Anvil()),
			current:  crds(kinds.Anvil()),
		},
		{
			name: "remove an unused CRD",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(),
				},
			},
			previous: crds(kinds.Anvil()),
		},
		{
			name: "remove an in-use CRD",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
			previous: crds(kinds.Anvil()),
			wantErr:  nonhierarchical.UnsupportedCRDRemovalError(fake.AnvilAtPath("anvil1.yaml")),
		},
	}

	for _, tc := range testCases {
		crv := CRDRemovalValidator(tc.previous, tc.current)
		t.Run(tc.name, func(t *testing.T) {

			err := crv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got CRDRemovalValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
