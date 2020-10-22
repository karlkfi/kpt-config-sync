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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestScopeValidator(t *testing.T) {
	scoper := discovery.NewScoper(map[schema.GroupKind]discovery.ScopeType{
		kinds.Role().GroupKind():        discovery.NamespaceScope,
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,
	}, true)

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
		nht.Pass("Kptfile", fake.KptFile("Kptfile", core.Namespace(""))),
		nht.Pass("Kptfile", fake.KptFile("Kptfile", core.Namespace("backend"))),
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

	nht.RunAll(t, nonhierarchical.ScopeValidator(metav1.NamespaceDefault, scoper), testCases)
}

func TestScopeValidator_AddsDefaultNamespace(t *testing.T) {
	scoper := discovery.NewScoper(map[schema.GroupKind]discovery.ScopeType{
		kinds.Role().GroupKind(): discovery.NamespaceScope,
	}, true)
	v := nonhierarchical.ScopeValidator(metav1.NamespaceDefault, scoper)

	r := fake.Role(core.Namespace(""))
	err := v.Validate([]ast.FileObject{r})
	if err != nil {
		t.Errorf("got Validate() = %v, want nil", err)
	} else if ns := r.GetNamespace(); ns != metav1.NamespaceDefault {
		t.Errorf("got metadata.namespace = %q, want %q", ns, metav1.NamespaceDefault)
	}
}

func TestScopeValidator_AddsSetNamespace(t *testing.T) {
	scoper := discovery.NewScoper(map[schema.GroupKind]discovery.ScopeType{
		kinds.Role().GroupKind(): discovery.NamespaceScope,
	}, true)
	v := nonhierarchical.ScopeValidator("shipping", scoper)

	r := fake.Role(core.Namespace(""))
	err := v.Validate([]ast.FileObject{r})
	if err != nil {
		t.Errorf("got Validate() = %v, want nil", err)
	} else if ns := r.GetNamespace(); ns != "shipping" {
		t.Errorf("got metadata.namespace = %q, want %q", ns, "shipping")
	}
}

func TestScopeValidator_LeavesClusterScopedBlank(t *testing.T) {
	scoper := discovery.NewScoper(map[schema.GroupKind]discovery.ScopeType{
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,
	}, true)
	v := nonhierarchical.ScopeValidator(metav1.NamespaceDefault, scoper)

	r := fake.ClusterRole(core.Namespace(""))
	err := v.Validate([]ast.FileObject{r})
	if err != nil {
		t.Errorf("got Validate() = %v, want nil", err)
	} else if ns := r.GetNamespace(); ns != "" {
		t.Errorf("got metadata.namespace = %q, want \"\"", ns)
	}
}
