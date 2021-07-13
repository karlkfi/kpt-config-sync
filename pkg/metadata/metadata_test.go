// Set the package name to `metadata_test` to avoid import cycles.
package metadata_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestHasConfigSyncMetadata(t *testing.T) {
	testcases := []struct {
		name string
		obj  client.Object
		want bool
	}{
		{
			name: "An object without Config Sync metadata",
			obj:  fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy")),
			want: false,
		},
		{
			name: "An object with the `OwningInventoryKey` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(metadata.OwningInventoryKey, "random-value")),
			want: true,
		},
		{
			name: "An object with the `LifecycleMutationAnnotation` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(metadata.LifecycleMutationAnnotation, "random-value")),
			want: true,
		},
		{
			name: "An object with the `client.lifecycle.config.k8s.io/others` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(metadata.LifecyclePrefix+"/others", "random-value")),
			want: false,
		},
		{
			name: "An object with the `ResourceManagementKey` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(metadata.ResourceManagementKey, metadata.ResourceManagementEnabled)),
			want: true,
		},
		{
			name: "An object with the `ResourceIDKey` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(metadata.ResourceIDKey, "random-value")),
			want: true,
		},
		{
			name: "An object with the `HNCManagedBy` annotation (random value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(metadata.HNCManagedBy, "random-value")),
			want: false,
		},
		{
			name: "An object with the `HNCManagedBy` annotation (correct value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(metadata.HNCManagedBy, configmanagement.GroupName)),
			want: true,
		},
		{
			name: "An object with the `DeclaredVersionLabel` label",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(metadata.DeclaredVersionLabel, "v1")),
			want: true,
		},
		{
			name: "An object with the `SystemLabel` label",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(metadata.SystemLabel, "random-value")),
			want: true,
		},
		{
			name: "An object with the `ManagedByKey` label (correct value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(metadata.ManagedByKey, metadata.ManagedByValue)),
			want: true,
		},
		{
			name: "An object with the `ManagedByKey` label (random value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(metadata.ManagedByKey, "random-value")),
			want: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := metadata.HasConfigSyncMetadata(tc.obj)
			if got != tc.want {
				t.Errorf("got HasConfigSyncMetadata() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRemoveConfigSyncMetadata(t *testing.T) {
	obj := fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
		syncertest.ManagementEnabled,
		core.Annotation(metadata.OwningInventoryKey, "random-value"),
		core.Annotation(metadata.LifecycleMutationAnnotation, "random-value"),
		core.Annotation(metadata.HNCManagedBy, configmanagement.GroupName),
		core.Label(metadata.DeclaredVersionLabel, "v1"),
		core.Label(metadata.SystemLabel, "random-value"),
		core.Label(metadata.ManagedByKey, metadata.ManagedByValue))
	updated := metadata.RemoveConfigSyncMetadata(obj)
	if !updated {
		t.Errorf("updated should be true")
	}
	labels := obj.GetLabels()
	if len(labels) > 0 {
		t.Errorf("labels should be empty, but got %v", labels)
	}

	annotations := obj.GetAnnotations()
	expectedAnnotation := map[string]string{
		metadata.LifecycleMutationAnnotation: "random-value",
	}
	if diff := cmp.Diff(annotations, expectedAnnotation); diff != "" {
		t.Errorf("Diff from the annotations is %s", diff)
	}

	updated = metadata.RemoveConfigSyncMetadata(obj)
	if updated {
		t.Errorf("the labels and annotations shouldn't be updated in this case")
	}
}
