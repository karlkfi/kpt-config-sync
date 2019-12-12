package nonhierarchical

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func crds(groupKinds ...schema.GroupKind) []*v1beta1.CustomResourceDefinition {
	var result []*v1beta1.CustomResourceDefinition
	for _, groupKind := range groupKinds {
		result = append(result, &v1beta1.CustomResourceDefinition{
			Spec: v1beta1.CustomResourceDefinitionSpec{
				Group: groupKind.Group,
				Names: v1beta1.CustomResourceDefinitionNames{
					Kind: groupKind.Kind,
				},
			},
		})
	}
	return result
}

func TestCRDRemovalValidator(t *testing.T) {
	testCases := []struct {
		name         string
		syncedCRDs   []*v1beta1.CustomResourceDefinition
		declaredCRDs []*v1beta1.CustomResourceDefinition
		objects      []ast.FileObject
		expectError  bool
	}{
		{
			name: "empty is fine",
		},
		{
			name:        "forbid removing CRD but keeping CR",
			syncedCRDs:  crds(kinds.Anvil().GroupKind()),
			objects:     []ast.FileObject{fake.AnvilAtPath("")},
			expectError: true,
		},
		{
			name:         "allow adding CRD and CR simultaneously",
			declaredCRDs: crds(kinds.Anvil().GroupKind()),
			objects:      []ast.FileObject{fake.AnvilAtPath("")},
		},
		{
			name:         "allow keeping CR with synced and declared CRD",
			syncedCRDs:   crds(kinds.Anvil().GroupKind()),
			declaredCRDs: crds(kinds.Anvil().GroupKind()),
			objects:      []ast.FileObject{fake.AnvilAtPath("")},
		},
		{
			name:    "allow keeping CR with CRD neither synced nor declared",
			objects: []ast.FileObject{fake.AnvilAtPath("")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := CRDRemovalValidator(tc.syncedCRDs, tc.declaredCRDs)

			errs := validator.Validate(tc.objects)

			if tc.expectError {
				if errs == nil {
					t.Fatal("expected error")
				}
			} else if errs != nil {
				t.Fatal("unexpected error", errs)
			}
		})
	}
}
