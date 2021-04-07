package applier

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestPartitionObjs(t *testing.T) {
	testcases := []struct {
		name          string
		objs          []client.Object
		enabledCount  int
		disabledCount int
	}{
		{
			name: "all managed objs",
			objs: []client.Object{
				fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"), syncertest.ManagementEnabled),
				fake.ServiceObject(core.Name("service"), core.Namespace("default"), syncertest.ManagementEnabled),
			},
			enabledCount:  2,
			disabledCount: 0,
		},
		{
			name: "all disabled objs",
			objs: []client.Object{
				fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"), syncertest.ManagementDisabled),
				fake.ServiceObject(core.Name("service"), core.Namespace("default"), syncertest.ManagementDisabled),
			},
			enabledCount:  0,
			disabledCount: 2,
		},
		{
			name: "mixed objs",
			objs: []client.Object{
				fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"), syncertest.ManagementEnabled),
				fake.ServiceObject(core.Name("service"), core.Namespace("default"), syncertest.ManagementDisabled),
			},
			enabledCount:  1,
			disabledCount: 1,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			enabled, disabled := partitionObjs(tc.objs)
			if len(enabled) != tc.enabledCount {
				t.Errorf("expected %d enabled objects, but got %d", tc.enabledCount, enabled)
			}
			if len(disabled) != tc.disabledCount {
				t.Errorf("expected %d disabled objects, but got %d", tc.disabledCount, enabled)
			}
		})
	}
}

func TestObjMetaFrom(t *testing.T) {
	d := fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))
	expected := object.ObjMetadata{
		Namespace: "default",
		Name:      "deploy",
		GroupKind: schema.GroupKind{
			Group: "apps",
			Kind:  "Deployment",
		},
	}
	actual := objMetaFrom(d)
	if actual != expected {
		t.Errorf("expected %v but got %v", expected, actual)
	}
}

func TestIDFrom(t *testing.T) {
	d := fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))
	meta := objMetaFrom(d)
	id := idFrom(meta)
	if id != core.IDOf(d) {
		t.Errorf("expected %v but got %v", core.IDOf(d), id)
	}
}

func TestRemoveFrom(t *testing.T) {
	testcases := []struct {
		name       string
		allObjMeta []object.ObjMetadata
		objs       []client.Object
		expected   []object.ObjMetadata
	}{
		{
			name: "toRemove is empty",
			allObjMeta: []object.ObjMetadata{
				objMetaFrom(fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))),
				objMetaFrom(fake.ServiceObject(core.Name("service"), core.Namespace("default"))),
			},
			objs: nil,
			expected: []object.ObjMetadata{
				objMetaFrom(fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))),
				objMetaFrom(fake.ServiceObject(core.Name("service"), core.Namespace("default"))),
			},
		},
		{
			name: "all toRemove are in the original list",
			allObjMeta: []object.ObjMetadata{
				objMetaFrom(fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))),
				objMetaFrom(fake.ServiceObject(core.Name("service"), core.Namespace("default"))),
			},
			objs: []client.Object{
				fake.ServiceObject(core.Name("service"), core.Namespace("default")),
			},
			expected: []object.ObjMetadata{
				objMetaFrom(fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))),
			},
		},
		{
			name: "some toRemove are not in the original list",
			allObjMeta: []object.ObjMetadata{
				objMetaFrom(fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))),
				objMetaFrom(fake.ServiceObject(core.Name("service"), core.Namespace("default"))),
			},
			objs: []client.Object{
				fake.ServiceObject(core.Name("service"), core.Namespace("default")),
				fake.ConfigMapObject(core.Name("cm"), core.Namespace("default")),
			},
			expected: []object.ObjMetadata{
				objMetaFrom(fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))),
			},
		},
		{
			name: "toRemove are the same as original objects",
			allObjMeta: []object.ObjMetadata{
				objMetaFrom(fake.DeploymentObject(core.Name("deploy"), core.Namespace("default"))),
				objMetaFrom(fake.ServiceObject(core.Name("service"), core.Namespace("default"))),
			},
			objs: []client.Object{
				fake.DeploymentObject(core.Name("deploy"), core.Namespace("default")),
				fake.ServiceObject(core.Name("service"), core.Namespace("default")),
			},
			expected: nil,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := removeFrom(tc.allObjMeta, tc.objs)
			if diff := cmp.Diff(tc.expected, actual, cmpopts.SortSlices(
				func(x, y object.ObjMetadata) bool { return x.String() < y.String() })); diff != "" {
				t.Errorf("%s: Diff of removeFrom is: %s", tc.name, diff)
			}
		})
	}
}

func TestRmoveConfigSyncLabelsAndAnnotations(t *testing.T) {
	obj := fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"), syncertest.ManagementEnabled, core.Annotation(OwningInventoryKey, "random-value"), core.Label(v1.ManagedByKey, v1.ManagedByValue))
	labels, annotations, updated := removeConfigSyncLabelsAndAnnotations(obj)
	if !updated {
		t.Errorf("updated should be true")
	}
	if len(labels) > 0 {
		t.Errorf("labels should be empty, but got %v", labels)
	}
	expectedAnnotation := map[string]string{v1.ResourceManagementKey: v1.ResourceManagementDisabled}
	if diff := cmp.Diff(annotations, expectedAnnotation); diff != "" {
		t.Errorf("Diff from the annotations is %s", diff)
	}

	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	_, _, updated = removeConfigSyncLabelsAndAnnotations(obj)
	if updated {
		t.Errorf("the labels and annotations shouldn't be updated in this case")
	}
}
