package resourcegroup

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configsync"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func fakeKptfile(labels, annotations map[string]string) *kptfile.KptFile {
	obj := &kptfile.KptFile{}
	obj.Inventory = &kptfile.Inventory{
		Name:        "test-rg",
		Namespace:   "test-namespace",
		Labels:      labels,
		Annotations: annotations,
	}
	return obj
}

func fakeResourceGroup(labels, annotations map[string]string, ids []ObjMetadata) *ResourceGroup {
	obj := &ResourceGroup{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   configsync.GroupName,
		Version: version,
		Kind:    kind,
	})
	obj.SetName("test-rg")
	obj.SetNamespace("test-namespace")
	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	obj.Spec.Descriptor = Descriptor{Type: application}
	obj.Spec.Resources = ids

	return obj
}

func fakeObjMetadata() []ObjMetadata {
	return []ObjMetadata{
		{
			Name:      "random-name",
			Namespace: "random-namespace",
			Group:     "group.a.io",
			Kind:      "KindA",
		},
		{
			Name:      "random-name",
			Namespace: "random-namespace",
			Group:     "group.b.io",
			Kind:      "KindB",
		},
	}
}

func TestGenerateResourceGroup(t *testing.T) {
	tcs := []struct {
		testName string
		kptFile  *kptfile.KptFile
		ids      []ObjMetadata
		expect   *ResourceGroup
	}{
		{
			testName: "empty ids",
			kptFile:  fakeKptfile(nil, nil),
			expect:   fakeResourceGroup(nil, nil, nil),
		},
		{
			testName: "empty ids with annotations",
			kptFile:  fakeKptfile(nil, map[string]string{"random-key": "random-value"}),
			expect:   fakeResourceGroup(nil, map[string]string{"random-key": "random-value"}, nil),
		},
		{
			testName: "empty ids with labels",
			kptFile:  fakeKptfile(map[string]string{"random-key": "random-value"}, nil),
			expect:   fakeResourceGroup(map[string]string{"random-key": "random-value"}, nil, nil),
		},
		{
			testName: "non empty ids",
			kptFile:  fakeKptfile(nil, nil),
			ids:      fakeObjMetadata(),
			expect:   fakeResourceGroup(nil, nil, fakeObjMetadata()),
		},
		{
			testName: "non empty ids with annotations",
			kptFile:  fakeKptfile(nil, map[string]string{"random-key": "random-value"}),
			ids:      fakeObjMetadata(),
			expect:   fakeResourceGroup(nil, map[string]string{"random-key": "random-value"}, fakeObjMetadata()),
		},
		{
			testName: "non empty ids with labels",
			kptFile:  fakeKptfile(map[string]string{"random-key": "random-value"}, nil),
			ids:      fakeObjMetadata(),
			expect:   fakeResourceGroup(map[string]string{"random-key": "random-value"}, nil, fakeObjMetadata()),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			actual := FromKptFile(tc.kptFile, tc.ids)
			if diff := cmp.Diff(tc.expect, actual, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
