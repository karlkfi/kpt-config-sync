package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func metaObject(m *metav1.ObjectMeta) ast.FileObject {
	d := fake.DeploymentObject()
	d.ObjectMeta = *m
	return ast.NewFileObject(d, cmpath.FromSlash("namespaces/foo/object.yaml"))
}

func TestDisallowedFieldsValidator(t *testing.T) {
	timeNow := metav1.Now()
	second := int64(1)

	testCases := []nht.ValidatorTestCase{
		nht.Pass("normal Role",
			fake.Role()),
		nht.Fail("replicaSet with ownerReference",
			metaObject(&metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{{Name: "some_deployment"}}})),
		nht.Fail("deployment with selfLink",
			metaObject(&metav1.ObjectMeta{SelfLink: "self_link"})),
		nht.Fail("deployment with uid",
			metaObject(&metav1.ObjectMeta{UID: "uid"})),
		nht.Fail("deployment with resourceVersion",
			metaObject(&metav1.ObjectMeta{ResourceVersion: "1"})),
		nht.Fail("deployment with generation",
			metaObject(&metav1.ObjectMeta{Generation: 1})),
		nht.Fail("deployment with creationTimestamp",
			metaObject(&metav1.ObjectMeta{CreationTimestamp: timeNow})),
		nht.Fail("deployment with deletionTimestamp",
			metaObject(&metav1.ObjectMeta{DeletionTimestamp: &timeNow})),
		nht.Fail("deployment with deletionGracePeriodSeconds",
			metaObject(&metav1.ObjectMeta{DeletionGracePeriodSeconds: &second})),
	}

	nht.RunAll(t, nonhierarchical.DisallowedFieldsValidator, testCases)
}
