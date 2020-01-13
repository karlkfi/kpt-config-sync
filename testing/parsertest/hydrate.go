package parsertest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/restmapper"
)

// TestCase represents a test case that runs AST hydration on a set of already-parsed files.
// These test cases run steps such as vetting that configuration is valid and performing Abstract
// Namespace inheritance.
//
// If error codes are expected, list them in Errors.
// Otherwise, fill in the expected hydrated hierarchy in Expected.
type TestCase struct {
	// Name is the name of the test case.
	Name string

	// Objects is the array of objects to run hydration and validation on.
	Objects []ast.FileObject

	// Expected is the resulting expected Root. Ignored if Errors is non-nil.
	Expected *namespaceconfig.AllConfigs

	// Errors is the errors the test case expects, if any.
	Errors []string

	// SyncedCRDs is the set of CRDs currently synced by ACM to the cluster.
	SyncedCRDs []*v1beta1.CustomResourceDefinition

	// ClusterName is the name of the cluster this test case is for.
	ClusterName string

	// Serverless is whether to ignore APIServer checks.
	Serverless bool
}

// ForCluster modifies this TestCase to be applied to a specific cluster.
func (tc TestCase) ForCluster(clusterName string) TestCase {
	tc.ClusterName = clusterName
	return tc
}

// DisableAPIServerChecks disables throwing errors when APIServer checks fail.
func (tc TestCase) DisableAPIServerChecks() TestCase {
	tc.Serverless = true
	return tc
}

// VetTest performs the sensible defaults for testing most parser vetting and hydrating behavior.
func VetTest(testCases ...TestCase) Test {
	return Test{
		DefaultObjects: func() []ast.FileObject {
			return []ast.FileObject{
				fake.Repo(),
			}
		},
		TestCases: testCases,
	}
}

// Test is a suite of tests to run on Parser's vetting and hydration functionality.
type Test struct {
	// NewParser returns a function which produces a Parser configured to hydrate a dry representation
	// of the hierarchy. This allows generating a new Parser for each test.
	NewParser func(t *testing.T) *filesystem.Parser

	// DefaultObjects is the list of objects implicitly included in every test case.
	DefaultObjects func() []ast.FileObject

	// TestCases is the list of test cases to run.
	TestCases []TestCase
}

// Success represents a test case which is expected to hydrate successfully.
func Success(name string, expected *namespaceconfig.AllConfigs, objects ...ast.FileObject) TestCase {
	return TestCase{
		Name:     name,
		Expected: expected,
		Objects:  objects,
	}
}

// Failure represents a test case which is expected to fail with a single error code.
func Failure(name string, err string, objects ...ast.FileObject) TestCase {
	return TestCase{
		Name:    name,
		Errors:  []string{err},
		Objects: objects,
	}
}

// Failures represents a test case which is expected to fail with multiple error codes.
func Failures(name string, errs []string, objects ...ast.FileObject) TestCase {
	return TestCase{
		Name:    name,
		Errors:  errs,
		Objects: objects,
	}
}

// fakeParserReader is a map from a path relative to a repo root to the objects
// contained in that directory for the test.
type fakeParserReader map[cmpath.Relative][]ast.FileObject

// Read implements Reader.
func (r fakeParserReader) Read(path cmpath.RootedPath) ([]ast.FileObject, status.MultiError) {
	return r[path.(cmpath.Relative)], nil
}

// CRDsToAPIGroupResources converts a list of CRDs to the corresponding APIGroupResources the
// server would return.
//
// As-is assumes each CRD is a different APIGroup. Don't bother fixing unless you need to test
// a case where you need to sync multiple CRDs for the same APIGroup.
func CRDsToAPIGroupResources(crds []*v1beta1.CustomResourceDefinition) []*restmapper.APIGroupResources {
	var result []*restmapper.APIGroupResources
	for _, syncedCRD := range crds {
		extraResource := &restmapper.APIGroupResources{
			Group: metav1.APIGroup{
				Name: syncedCRD.Spec.Group,
				PreferredVersion: metav1.GroupVersionForDiscovery{
					GroupVersion: syncedCRD.Spec.Group + "/" + syncedCRD.Spec.Versions[0].Name,
				},
			},
		}

		extraResource.VersionedResources = make(map[string][]metav1.APIResource)
		for _, version := range syncedCRD.Spec.Versions {
			extraResource.Group.Versions = append(extraResource.Group.Versions,
				metav1.GroupVersionForDiscovery{
					GroupVersion: syncedCRD.Spec.Group + "/" + version.Name,
					Version:      version.Name,
				},
			)
			extraResource.VersionedResources[version.Name] = []metav1.APIResource{
				{
					Name: syncedCRD.Spec.Names.Plural,
					SingularName: syncedCRD.Spec.Names.Singular,
					Namespaced: !(syncedCRD.Spec.Scope == v1beta1.ClusterScoped),
					Group: syncedCRD.Spec.Group,
					Version: version.Name,
					Kind: syncedCRD.Spec.Names.Kind,
				},
			}
		}

		result = append(result, extraResource)
	}
	return result
}

// newTestParser creates a parser that processes the passed FileObjects.
func newTestParser(t *testing.T, objects []ast.FileObject, syncedCRDs []*v1beta1.CustomResourceDefinition) *filesystem.Parser {
	f := fstesting.NewTestClientGetter(CRDsToAPIGroupResources(syncedCRDs))

	root := cmpath.Root{}
	flatRoot := treetesting.BuildFlatTree(t, objects...)
	reader := fakeParserReader{
		root.Join(cmpath.FromSlash(repo.SystemDir)):          flatRoot.SystemObjects,
		root.Join(cmpath.FromSlash(repo.ClusterRegistryDir)): flatRoot.ClusterRegistryObjects,
		root.Join(cmpath.FromSlash(repo.ClusterDir)):         flatRoot.ClusterObjects,
		root.Join(cmpath.FromSlash(repo.NamespacesDir)):      flatRoot.NamespaceObjects,
	}

	return filesystem.NewParser(root, reader, f)
}

// RunAll runs each unit test.
func (pt Test) RunAll(t *testing.T) {
	t.Helper()

	for _, tc := range pt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			objects := tc.Objects
			if pt.DefaultObjects != nil {
				// Add default objects, if they are defined for this test suite.
				objects = append(pt.DefaultObjects(), objects...)
			}
			parser := newTestParser(t, objects, tc.SyncedCRDs)

			getSyncedCRDs := func() ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
				return tc.SyncedCRDs, nil
			}

			fileObjects, errs := parser.Parse(tc.ClusterName, !tc.Serverless, getSyncedCRDs)
			actual := namespaceconfig.NewAllConfigs(visitortesting.ImportToken, metav1.Time{}, fileObjects)

			if tc.Errors != nil || errs != nil {
				// We either expected an error, or experienced an unexpected error.
				vettesting.ExpectErrors(tc.Errors, errs, t)
				return
			}

			if tc.Expected == nil {
				// Make error messages for expected successes more helpful when writing tests.
				tc.Expected = &namespaceconfig.AllConfigs{}
			}
			if tc.Expected.ClusterConfig == nil {
				// Assume a default empty and valid ClusterConfig if none specified.
				tc.Expected.ClusterConfig = fake.ClusterConfigObject()
			}
			if tc.Expected.CRDClusterConfig == nil {
				// Assume a default empty and valid CRDClusterConfig if none specified.
				tc.Expected.CRDClusterConfig = fake.CRDClusterConfigObject()
			}
			if tc.Expected.NamespaceConfigs == nil {
				// Make NamespaceConfig errors more helpful when writing tests.
				tc.Expected.NamespaceConfigs = map[string]v1.NamespaceConfig{}
			}

			if diff := cmp.Diff(tc.Expected, actual, cmpopts.EquateEmpty(), resourcequota.ResourceQuantityEqual()); diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}
