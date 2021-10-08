package hydrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
	"sigs.k8s.io/cli-utils/pkg/common"
)

func TestPreventDeletion(t *testing.T) {
	objs := &objects.Raw{
		Objects: []ast.FileObject{
			fake.ClusterRoleAtPath("cluster/clusterrole.yaml", core.Name("reader")),
			fake.Namespace("namespaces/default"),
			fake.Namespace("namespaces/kube-system"),
			fake.Namespace("namespaces/kube-public"),
			fake.Namespace("namespaces/kube-node-lease"),
			fake.Namespace("namespaces/gatekeeper-system"),
			fake.Namespace("namespaces/bookstore"),
		},
	}
	want := &objects.Raw{
		Objects: []ast.FileObject{
			fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
				core.Name("reader")),
			fake.Namespace("namespaces/default", core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)),
			fake.Namespace("namespaces/kube-system", core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)),
			fake.Namespace("namespaces/kube-public", core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)),
			fake.Namespace("namespaces/kube-node-lease", core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)),
			fake.Namespace("namespaces/gatekeeper-system", core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)),
			fake.Namespace("namespaces/bookstore"),
		},
	}

	if err := PreventDeletion(objs); err != nil {
		t.Errorf("Got PreventDeletion() error %v, want nil", err)
	}
	if diff := cmp.Diff(want, objs, ast.CompareFileObject); diff != "" {
		t.Error(diff)
	}
}
