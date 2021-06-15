package metadata

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/webhook/configuration"
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
			name: "An object with the `constants.OwningInventoryKey` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(constants.OwningInventoryKey, "random-value")),
			want: true,
		},
		{
			name: "An object with the `constants.LifecycleMutationAnnotation` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(constants.LifecycleMutationAnnotation, "random-value")),
			want: true,
		},
		{
			name: "An object with the `client.lifecycle.config.k8s.io/others` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(constants.LifecyclePrefix+"/others", "random-value")),
			want: false,
		},
		{
			name: "An object with the `v1.ResourceManagementKey` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			want: true,
		},
		{
			name: "An object with the `constants.ResourceIDKey` annotation",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(constants.ResourceIDKey, "random-value")),
			want: true,
		},
		{
			name: "An object with the `hnc.AnnotationKeyV1A2` annotation (random value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(hnc.AnnotationKeyV1A2, "random-value")),
			want: false,
		},
		{
			name: "An object with the `hnc.AnnotationKeyV1A2` annotation (correct value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Annotation(hnc.AnnotationKeyV1A2, configmanagement.GroupName)),
			want: true,
		},
		{
			name: "An object with the `configuration.DeclaredVersionLabel` label",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(configuration.DeclaredVersionLabel, "v1")),
			want: true,
		},
		{
			name: "An object with the `v1.SystemLabel` label",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(v1.SystemLabel, "random-value")),
			want: true,
		},
		{
			name: "An object with the `v1.ManagedByKey` label (correct value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue)),
			want: true,
		},
		{
			name: "An object with the `v1.ManagedByKey` label (random value)",
			obj: fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
				core.Label(v1.ManagedByKey, "random-value")),
			want: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := HasConfigSyncMetadata(tc.obj)
			if got != tc.want {
				t.Errorf("got HasConfigSyncMetadata() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRemoveConfigSyncMetadata(t *testing.T) {
	obj := fake.UnstructuredObject(kinds.Deployment(), core.Name("deploy"),
		syncertest.ManagementEnabled,
		core.Annotation(constants.OwningInventoryKey, "random-value"),
		core.Annotation(constants.LifecycleMutationAnnotation, "random-value"),
		core.Annotation(hnc.AnnotationKeyV1A2, configmanagement.GroupName),
		core.Label(configuration.DeclaredVersionLabel, "v1"),
		core.Label(v1.SystemLabel, "random-value"),
		core.Label(v1.ManagedByKey, v1.ManagedByValue))
	updated := RemoveConfigSyncMetadata(obj)
	if !updated {
		t.Errorf("updated should be true")
	}
	labels := obj.GetLabels()
	if len(labels) > 0 {
		t.Errorf("labels should be empty, but got %v", labels)
	}

	annotations := obj.GetAnnotations()
	expectedAnnotation := map[string]string{
		constants.LifecycleMutationAnnotation: "random-value",
	}
	if diff := cmp.Diff(annotations, expectedAnnotation); diff != "" {
		t.Errorf("Diff from the annotations is %s", diff)
	}

	updated = RemoveConfigSyncMetadata(obj)
	if updated {
		t.Errorf("the labels and annotations shouldn't be updated in this case")
	}
}
