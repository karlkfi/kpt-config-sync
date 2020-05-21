package nonhierarchical_test

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScopeValidator(t *testing.T) {
	scoper := discovery.Scoper{
		kinds.Role().GroupKind():        discovery.NamespaceScope,
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,
	}

	testCases := []nht.ValidatorTestCase{
		nht.Pass("Namespace-scoped object with metadata.namespace",
			fake.Role(core.Namespace("backend")),
		),
		nht.Pass("Namespace-scoped object without metadata.namespace",
			fake.Role(core.Namespace("")),
		),
		nht.Pass("Namespace-scoped object without metadata.namespace with namespace-selector",
			fake.Role(core.Namespace(""), core.Annotation(v1.NamespaceSelectorAnnotationKey, "value")),
		),
		nht.Fail("Namespace-scoped object with metadata.namespace and namespace-selector",
			fake.Role(core.Namespace("backend"), core.Annotation(v1.NamespaceSelectorAnnotationKey, "value")),
		),
		nht.Fail("Cluster-scoped object with metadata.namespace",
			fake.ClusterRole(core.Namespace("backend")),
		),
		nht.Pass("Cluster-scoped object with metadata.namespace",
			fake.ClusterRole(core.Namespace("")),
		),
		nht.Fail("Unknown type with metadata.namespace",
			fake.NamespaceSelector(core.Namespace("backend")),
		),
		nht.Fail("Unknown type without metadata.namespace",
			fake.NamespaceSelector(core.Namespace("")),
		),
	}

	nht.RunAll(t, nonhierarchical.ScopeValidator(scoper), testCases)
}

func TestScopeValidator_AddsDefaultNamespace(t *testing.T) {
	scoper := discovery.Scoper{
		kinds.Role().GroupKind(): discovery.NamespaceScope,
	}
	v := nonhierarchical.ScopeValidator(scoper)

	r := fake.Role(core.Namespace(""))
	err := v.Validate([]ast.FileObject{r})
	if err != nil {
		t.Errorf("got Validate() = %v, want nil", err)
	} else if ns := r.GetNamespace(); ns != metav1.NamespaceDefault {
		t.Errorf("got metadata.namespace = %q, want %q", ns, metav1.NamespaceDefault)
	}
}

func TestScopeValidator_LeavesClusterScopedBlank(t *testing.T) {
	scoper := discovery.Scoper{
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,
	}
	v := nonhierarchical.ScopeValidator(scoper)

	r := fake.ClusterRole(core.Namespace(""))
	err := v.Validate([]ast.FileObject{r})
	if err != nil {
		t.Errorf("got Validate() = %v, want nil", err)
	} else if ns := r.GetNamespace(); ns != "" {
		t.Errorf("got metadata.namespace = %q, want \"\"", ns)
	}
}
