package core_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRemarshalToStructured(t *testing.T) {
	testcases := []struct {
		name string
		u    *unstructured.Unstructured
		obj  runtime.Object
	}{
		{
			name: "v1alpha1 RepoSync",
			u:    fake.UnstructuredObject(kinds.RepoSync(), core.Name(constants.RepoSyncName), core.Namespace("test"), core.Annotations(nil), core.Labels(nil)),
			obj:  fake.RepoSyncObject(core.Namespace("test")),
		},
		{
			name: "v1beta1 RepoSync",
			u:    fake.UnstructuredObject(kinds.RepoSyncV1Beta1(), core.Name(constants.RepoSyncName), core.Namespace("test"), core.Annotations(nil), core.Labels(nil)),
			obj:  fake.RepoSyncObjectV1Beta1(core.Namespace("test")),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := core.RemarshalToStructured(tc.u)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(actual, tc.obj); diff != "" {
				t.Error(diff)
			}
		})
	}
}
