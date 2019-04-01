package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func metaObject(gvk schema.GroupVersionKind, m *metav1.ObjectMeta) ast.FileObject {
	o := asttesting.NewFakeObject(gvk).WithMeta(m)
	return ast.NewFileObject(o, cmpath.FromSlash("namespaces/foo/object.yaml"))
}

func TestDisallowedFieldsValidator(t *testing.T) {
	timeNow := metav1.Now()
	second := int64(1)

	test := vt.ObjectValidatorTest{
		Validator: NewDisallowedFieldsValidator,
		ErrorCode: vet.IllegalFieldsInConfigErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "deployment without ownerReference",
				Object: fake.Deployment("namespaces/foo/deployment.yaml"),
			},
			{
				Name:       "replicaSet with ownerReference",
				Object:     fake.Build(kinds.ReplicaSet(), object.OwnerReference("some_deployment", "some_uid", kinds.Deployment())),
				ShouldFail: true,
			},
			{
				Name:       "deployment with selfLink",
				Object:     metaObject(kinds.Deployment(), &metav1.ObjectMeta{SelfLink: "self_link"}),
				ShouldFail: true,
			},
			{
				Name:       "deployment with uid",
				Object:     metaObject(kinds.Deployment(), &metav1.ObjectMeta{UID: "uid"}),
				ShouldFail: true,
			},
			{
				Name:       "deployment with resourceVersion",
				Object:     metaObject(kinds.Deployment(), &metav1.ObjectMeta{ResourceVersion: "1"}),
				ShouldFail: true,
			},
			{
				Name:       "deployment with generation",
				Object:     metaObject(kinds.Deployment(), &metav1.ObjectMeta{Generation: 1}),
				ShouldFail: true,
			},
			{
				Name:       "deployment with creationTimestamp",
				Object:     metaObject(kinds.Deployment(), &metav1.ObjectMeta{CreationTimestamp: timeNow}),
				ShouldFail: true,
			},
			{
				Name:       "deployment with deletionTimestamp",
				Object:     metaObject(kinds.Deployment(), &metav1.ObjectMeta{DeletionTimestamp: &timeNow}),
				ShouldFail: true,
			},
			{
				Name:       "deployment with deletionGracePeriodSeconds",
				Object:     metaObject(kinds.Deployment(), &metav1.ObjectMeta{DeletionGracePeriodSeconds: &second}),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
