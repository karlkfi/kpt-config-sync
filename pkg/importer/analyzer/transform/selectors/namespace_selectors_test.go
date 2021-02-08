package selectors

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	nsName         = "frontend"
	nsSelectorName = "sre-supported"

	selectorDir    = "namespaces/foo"
	nonSelectorDir = "namespaces/bar"

	sreSupport = "sre-support"
)

func TestResolveHierarchicalNamespaceSelectors(t *testing.T) {
	namespaceSelectorObject := fake.NamespaceSelectorObject(core.Name(nsSelectorName))
	namespaceSelectorObject.Spec.Selector.MatchLabels = map[string]string{
		sreSupport: "true",
	}
	namespaceSelector := fake.FileObject(namespaceSelectorObject, selectorDir+"/selector.yaml")

	testCases := []struct {
		name               string
		namespaceParentDir string
		namespaceLabels    map[string]string
		objectAnnotations  map[string]string
		shouldKeep         bool
		shouldFail         bool
	}{
		// Trivial cases
		{
			name:               "Object without selector and Namespace without labels is kept",
			namespaceParentDir: selectorDir,
			shouldKeep:         true,
		},
		{
			name:               "Object outside selector dir without selector and Namespace without labels is kept",
			namespaceParentDir: nonSelectorDir,
			shouldKeep:         true,
		},
		{
			name:               "Object without selector and Namespace with labels is kept",
			namespaceParentDir: selectorDir,
			namespaceLabels:    map[string]string{sreSupport: "true"},
			shouldKeep:         true,
		},
		{
			name:               "Object without selector and Namespace with wrong label is not kept",
			namespaceParentDir: selectorDir,
			namespaceLabels:    map[string]string{sreSupport: "false"},
			shouldKeep:         true,
		},
		{
			name:               "Object with selector and Namespace without labels is not kept",
			namespaceParentDir: selectorDir,
			objectAnnotations:  map[string]string{v1.NamespaceSelectorAnnotationKey: nsSelectorName},
		},
		{
			name:               "Object and Namespace with labels is kept",
			namespaceParentDir: selectorDir,
			namespaceLabels:    map[string]string{sreSupport: "true"},
			objectAnnotations:  map[string]string{v1.NamespaceSelectorAnnotationKey: nsSelectorName},
			shouldKeep:         true,
		},
		// Error conditions
		{
			name:               "Object references selector outside dir causes error",
			namespaceParentDir: nonSelectorDir,
			objectAnnotations:  map[string]string{v1.NamespaceSelectorAnnotationKey: nsSelectorName},
			shouldFail:         true,
		},
		{
			name:               "Object references non-existent selector",
			namespaceParentDir: selectorDir,
			objectAnnotations:  map[string]string{v1.NamespaceSelectorAnnotationKey: "undeclared"},
			shouldFail:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namespace := fake.Namespace(tc.namespaceParentDir+"/"+nsName, core.Labels(tc.namespaceLabels))
			object := fake.RoleAtPath(tc.namespaceParentDir+"/role.yaml", core.Namespace(nsName), core.Annotations(tc.objectAnnotations))

			objects := []ast.FileObject{
				namespace,
				namespaceSelector,
				object,
			}

			actual, errs := ResolveHierarchicalNamespaceSelectors(objects)
			if tc.shouldFail {
				vettesting.ExpectErrors([]string{ObjectHasUnknownSelectorCode}, errs, t)
				return
			} else if errs != nil {
				vettesting.ExpectErrors([]string{}, errs, t)
			}

			expected := []ast.FileObject{namespace}
			if tc.shouldKeep {
				expected = append(expected, object)
			}

			if diff := cmp.Diff(expected, actual, ast.CompareFileObject); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestResolveHierarchicalNamespaceSelectors_InvalidSelector(t *testing.T) {
	namespaceSelectorObject := fake.NamespaceSelectorObject(core.Name(nsSelectorName))
	namespaceSelectorObject.Spec.Selector.MatchLabels = map[string]string{
		sreSupport: "xin true",
	}
	namespaceSelector := fake.FileObject(namespaceSelectorObject, selectorDir+"/selector.yaml")

	_, errs := ResolveHierarchicalNamespaceSelectors([]ast.FileObject{namespaceSelector})

	vettesting.ExpectErrors([]string{InvalidSelectorErrorCode}, errs, t)
}
