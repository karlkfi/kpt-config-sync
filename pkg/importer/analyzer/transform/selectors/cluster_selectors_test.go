package selectors

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
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

// withClusterSelector modifies a FileObject to have a cluster-selector annotation
// referencing the ClusterSelector named "name".
func withClusterSelector(name string) core.MetaMutator {
	return core.Annotation(v1.ClusterSelectorAnnotationKey, name)
}

var (
	withProdClusterSelector    = withClusterSelector(prodClusterSelectorName)
	withDevClusterSelector     = withClusterSelector(devClusterSelectorName)
	withUnknownClusterSelector = withClusterSelector("unknown")
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
			testName:    "keeps object in Namespace with active ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo", withProdClusterSelector),
				fake.Role(core.Namespace("foo")),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/foo", withProdClusterSelector),
				fake.Role(core.Namespace("foo")),
			},
		},
		{
			testName:    "discards Namespace with inactive ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo", withDevClusterSelector),
				fake.Role(core.Namespace("foo")),
			},
		},
		// Cluster-specific behavior
		{
			testName:    "keeps prod object in prod environment",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdClusterSelector),
			},
			expected: []ast.FileObject{
				fake.Role(withProdClusterSelector),
			},
		},
		{
			testName:    "discards dev object in prod environment",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevClusterSelector),
			},
		},
		{
			testName:    "discards prod object in dev environment",
			clusterName: devClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdClusterSelector),
			},
		},
		{
			testName:    "keeps dev object in dev environment",
			clusterName: devClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevClusterSelector),
			},
			expected: []ast.FileObject{
				fake.Role(withDevClusterSelector),
			},
		},
		{
			testName:    "discards prod object in unknown cluster",
			clusterName: unknownClusterName,
			objects: []ast.FileObject{
				fake.Role(withProdClusterSelector),
			},
		},
		{
			testName:    "discards dev object in unknown cluster",
			clusterName: unknownClusterName,
			objects: []ast.FileObject{
				fake.Role(withDevClusterSelector),
			},
		},
		// Error conditions
		{
			testName:    "error if missing ClusterSelector",
			clusterName: prodClusterName,
			objects: []ast.FileObject{
				fake.Role(withUnknownClusterSelector),
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

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
