package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
)

func metaObject(gvk schema.GroupVersionKind, m *metav1.ObjectMeta) ast.FileObject {
	o := asttesting.NewFakeObject(gvk).WithMeta(m)
	return ast.NewFileObject(o, cmpath.FromSlash("namespaces/foo/object.yaml"))
}

func TestDisallowedFieldsValidator(t *testing.T) {
	timeNow := metav1.Now()
	second := int64(1)

	testCases := []nht.ValidatorTestCase{
		nht.Pass("normal Role",
			fake.Role()),
		nht.Fail("replicaSet with ownerReference",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{{Name: "some_deployment"}}})),
		nht.Fail("deployment with selfLink",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{SelfLink: "self_link"})),
		nht.Fail("deployment with uid",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{UID: "uid"})),
		nht.Fail("deployment with resourceVersion",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{ResourceVersion: "1"})),
		nht.Fail("deployment with generation",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{Generation: 1})),
		nht.Fail("deployment with creationTimestamp",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{CreationTimestamp: timeNow})),
		nht.Fail("deployment with deletionTimestamp",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{DeletionTimestamp: &timeNow})),
		nht.Fail("deployment with deletionGracePeriodSeconds",
			metaObject(kinds.Deployment(), &metav1.ObjectMeta{DeletionGracePeriodSeconds: &second})),
	}

	nht.RunAll(t, nonhierarchical.DisallowedFieldsValidator, testCases)
}
