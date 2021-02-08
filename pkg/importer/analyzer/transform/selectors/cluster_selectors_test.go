package selectors

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	prodClusterName    = "prod-cluster"
	devClusterName     = "dev-cluster"
	unknownClusterName = "unknown-cluster"

	prodClusterSelectorName = "prod-selector"
	devClusterSelectorName  = "dev-selector"

	environmentLabel = "environment"
	prodEnvironment  = "prod"
	devEnvironment   = "dev"
)

var (
	prodCluster = cluster(prodClusterName, environmentLabel, prodEnvironment)
	devCluster  = cluster(devClusterName, environmentLabel, devEnvironment)

	prodSelector = clusterSelector(prodClusterSelectorName, environmentLabel, prodEnvironment)
	devSelector  = clusterSelector(devClusterSelectorName, environmentLabel, devEnvironment)
)

// clusterSelector creates a FileObject containing a ClusterSelector named "name",
// which matches Cluster objects with label "label" set to "value".
func clusterSelector(name, label, value string) ast.FileObject {
	cs := fake.ClusterSelectorObject(core.Name(name))
	cs.Spec.Selector.MatchLabels = map[string]string{label: value}
	return fake.FileObject(cs, fmt.Sprintf("clusterregistry/cs-%s.yaml", name))
}

// cluster creates a FileObject containing a Cluster named "name", with label "label"
// set to "value".
func cluster(name, label, value string) ast.FileObject {
	return fake.Cluster(core.Name(name), core.Label(label, value))
}

// withLegacyClusterSelector modifies a FileObject to have a legacy cluster-selector annotation
// referencing the ClusterSelector named "name".
func withLegacyClusterSelector(name string) core.MetaMutator {
	return core.Annotation(v1.LegacyClusterSelectorAnnotationKey, name)
}

// withInlineClusterNameSelector modifies a FileObject to have an inline cluster-selector annotation
// referencing the cluster matched with the labelSelector.
func withInlineClusterNameSelector(clusters string) core.MetaMutator {
	return core.Annotation(v1alpha1.ClusterNameSelectorAnnotationKey, clusters)
}

var (
	withProdLegacyClusterSelector    = withLegacyClusterSelector(prodClusterSelectorName)
	withDevLegacyClusterSelector     = withLegacyClusterSelector(devClusterSelectorName)
	withUnknownLegacyClusterSelector = withLegacyClusterSelector("unknown")
	withProdInlineMatchLabels        = withInlineClusterNameSelector(prodClusterName)
	withDevInlineMatchLabels         = withInlineClusterNameSelector(devClusterName)
)

func TestResolveClusterSelectors(t *testing.T) {
	testCases := []struct {
		testName       string
		clusterName    string
		objects        []ast.FileObject
		expected       []ast.FileObject
		expectedErrors []string
	}{
		// Trivial cases
		{
			testName:    "empty does nothing",
			clusterName: prodClusterName,
		},
		{
			testName:    "keeps object with no ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(),
			},
			expected: []ast.FileObject{
				fake.Role(),
			},
		},
		// Namespace inheritance
		{
			testName:    "keeps object in Namespace with no ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo"),
				fake.Role(core.Namespace("foo")),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/foo"),
				fake.Role(core.Namespace("foo")),
			},
		},
		{
			testName:    "keeps object in Namespace with active legacy ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo", withProdLegacyClusterSelector),
				fake.Role(core.Namespace("foo")),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/foo", withProdLegacyClusterSelector),
				fake.Role(core.Namespace("foo")),
			},
		},
		{
			testName:    "discards Namespace with inactive legacy ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo", withDevLegacyClusterSelector),
				fake.Role(core.Namespace("foo")),
			},
		},
		{
			testName:    "keeps object in Namespace with active inline cluster-name-selector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo", withProdInlineMatchLabels),
				fake.Role(core.Namespace("foo")),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/foo", withProdInlineMatchLabels),
				fake.Role(core.Namespace("foo")),
			},
		},
		{
			testName:    "discards Namespace with with inactive inline cluster-name-selector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo", withDevInlineMatchLabels),
				fake.Role(core.Namespace("foo")),
			},
		},
		// Cluster-specific behavior
		{
			testName:    "keeps prod object in prod environment with legacy ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdLegacyClusterSelector),
			},
			expected: []ast.FileObject{
				fake.Role(withProdLegacyClusterSelector),
			},
		},
		{
			testName:    "keeps prod object in prod environment with inline cluster-name-selector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdInlineMatchLabels),
			},
			expected: []ast.FileObject{
				fake.Role(withProdInlineMatchLabels),
			},
		},
		{
			testName:    "discards dev object in prod environment with legacy ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevLegacyClusterSelector),
			},
		},
		{
			testName:    "discards dev object in prod environment with inline cluster-name-selector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevInlineMatchLabels),
			},
		},
		{
			testName:    "discards prod object in dev environment with legacy ClusterSelector",
			clusterName: devClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdLegacyClusterSelector),
			},
		},
		{
			testName:    "discards prod object in dev environment with inline cluster-name-selector",
			clusterName: devClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdInlineMatchLabels),
			},
		},
		{
			testName:    "keeps dev object in dev environment with legacy ClusterSelector",
			clusterName: devClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevLegacyClusterSelector),
			},
			expected: []ast.FileObject{
				fake.Role(withDevLegacyClusterSelector),
			},
		},
		{
			testName:    "keeps dev object in dev environment with inline cluster-name-selector",
			clusterName: devClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevInlineMatchLabels),
			},
			expected: []ast.FileObject{
				fake.Role(withDevInlineMatchLabels),
			},
		},
		{
			testName:    "discards prod object in unknown cluster with legacy ClusterSelector",
			clusterName: unknownClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdLegacyClusterSelector),
			},
		},
		{
			testName:    "discards prod object in unknown cluster with inline cluster-name-selector",
			clusterName: unknownClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdInlineMatchLabels),
			},
		},
		{
			testName:    "discards dev object in unknown cluster with legacy ClusterSelector",
			clusterName: unknownClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevLegacyClusterSelector),
			},
		},
		{
			testName:    "discards dev object in unknown cluster with inline cluster-name-selector",
			clusterName: unknownClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevInlineMatchLabels),
			},
		},
		// Various inline cluster-name-selectors
		{
			testName:    "keeps prod object if clusterName in the target clusters",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector(fmt.Sprintf("%s,%s", prodClusterName, devClusterName))),
			},
			expected: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector(fmt.Sprintf("%s,%s", prodClusterName, devClusterName))),
			},
		},
		{
			testName:    "discards prod object if clusterName not in the target clusters",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector("dev,staging,others")),
			},
		},
		{
			testName:    "keeps prod object if clusterName in the target clusters (with space)",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector(fmt.Sprintf("%s, %s", devClusterName, prodClusterName))),
			},
			expected: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector(fmt.Sprintf("%s, %s", devClusterName, prodClusterName))),
			},
		},
		{
			testName:    "discards prod object if target clusters are empty",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector("")),
			},
		},
		{
			testName:    "discards prod object if target clusters include empty clusters",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector("a,,b")),
			},
		},
		{
			testName:    "discards object if clusterName and target clusters are empty",
			clusterName: "",
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector("")),
			},
		},
		{
			testName:    "discards object if clusterName is empty and target clusters do not include empty clusters",
			clusterName: "",
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector("a,b,c")),
			},
		},
		{
			testName:    "discards object if clusterName is empty and target clusters include empty clusters",
			clusterName: "",
			objects: []ast.FileObject{
				fake.Role(withInlineClusterNameSelector("a,,b")),
			},
		},
		// Error conditions
		{
			testName:    "error if both legacy cluster-selector annotation and inline annotation are specified",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdLegacyClusterSelector, withProdInlineMatchLabels),
			},
			expectedErrors: []string{ClusterSelectorAnnotationConflictErrorCode},
		},
		{
			testName:    "error if missing ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withUnknownLegacyClusterSelector),
			},
			expectedErrors: []string{ObjectHasUnknownSelectorCode},
		},
		{
			testName:    "error if invalid ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				clusterSelector("invalid", "environment", "xin prod"),
			},
			expectedErrors: []string{InvalidSelectorErrorCode},
		},
		{
			testName:    "error if empty ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.ClusterSelector(core.Name("empty")),
			},
			expectedErrors: []string{InvalidSelectorErrorCode},
		},
	}

	// objects included in every test
	baseObjects := []ast.FileObject{
		prodCluster,
		devCluster,
		prodSelector,
		devSelector,
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			fileObjects := append(baseObjects, tc.objects...)
			actual, err := ResolveClusterSelectors(tc.clusterName, fileObjects)

			vettesting.ExpectErrors(tc.expectedErrors, err, t)
			if tc.expectedErrors != nil {
				return
			}

			if diff := cmp.Diff(tc.expected, actual, ast.CompareFileObject); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
