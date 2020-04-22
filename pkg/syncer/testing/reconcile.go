package testing

import (
	"testing"
	"time"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
)

// Token is a test sync token.
const Token = "b38239ea8f58eaed17af6734bd6a025eeafccda1"

var (
	// Converter is an unstructured.Unstructured converter used for testing.
	Converter = runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())

	// ManagementEnabled sets management labels and annotations on the object.
	ManagementEnabled core.MetaMutator = func(obj core.Object) {
		core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
		core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
	}
	// ManagementDisabled sets the management disabled annotation on the object.
	ManagementDisabled = core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)
	// ManagementInvalid sets an invalid management annotation on the object.
	ManagementInvalid = core.Annotation(v1.ResourceManagementKey, "invalid")
	// TokenAnnotation sets the sync token annotation on the object
	TokenAnnotation = core.Annotation(v1.SyncTokenAnnotationKey, Token)
)

// ToUnstructured converts the object to an unstructured.Unstructured.
func toUnstructured(t *testing.T, converter runtime.UnstructuredConverter, obj runtime.Object) *unstructured.Unstructured {
	if obj == nil {
		return &unstructured.Unstructured{}
	}
	u, err := converter.ToUnstructured(obj)
	// We explicitly remove the status field from objects during reconcile. So,
	// we need to do the same for test objects we convert to unstructured.Unstructured.
	unstructured.RemoveNestedField(u, "status")
	if err != nil {
		t.Fatalf("could not convert to unstructured type: %#v", obj)
	}
	return &unstructured.Unstructured{Object: u}
}

// ToUnstructuredList converts the objects to an unstructured.UnstructedList.
func ToUnstructuredList(t *testing.T, converter runtime.UnstructuredConverter, objs ...runtime.Object) []*unstructured.Unstructured {
	result := make([]*unstructured.Unstructured, len(objs))
	for i, obj := range objs {
		result[i] = toUnstructured(t, converter, obj)
	}
	return result
}

// ClusterConfigImportToken adds an import token to a ClusterConfig.
func ClusterConfigImportToken(t string) fake.ClusterConfigMutator {
	return func(cc *v1.ClusterConfig) {
		cc.Spec.Token = t
	}
}

// ClusterConfigSyncTime adds a SyncTime to a ClusterConfig.
func ClusterConfigSyncTime() fake.ClusterConfigMutator {
	return func(cc *v1.ClusterConfig) {
		cc.Status.SyncTime = Now()
	}
}

// ClusterConfigSyncToken adds a sync token to a ClusterConfig.
func ClusterConfigSyncToken() fake.ClusterConfigMutator {
	return func(cc *v1.ClusterConfig) {
		cc.Status.Token = Token
	}
}

// NamespaceConfigImportToken adds an import token to a Namespace Config.
func NamespaceConfigImportToken(t string) fake.NamespaceConfigMutator {
	return func(nc *v1.NamespaceConfig) {
		nc.Spec.Token = t
	}
}

// NamespaceConfigSyncTime adds a sync time to a Namespace Config.
func NamespaceConfigSyncTime() fake.NamespaceConfigMutator {
	return func(nc *v1.NamespaceConfig) {
		nc.Status.SyncTime = Now()
	}
}

// NamespaceConfigSyncToken adds a sync token to a Namespace Config.
func NamespaceConfigSyncToken() fake.NamespaceConfigMutator {
	return func(nc *v1.NamespaceConfig) {
		nc.Status.Token = Token
	}
}

// MarkForDeletion marks a NamespaceConfig with an intent to be delete
func MarkForDeletion() fake.NamespaceConfigMutator {
	return func(nc *v1.NamespaceConfig) {
		nc.Spec.DeleteSyncedTime = metav1.Now()
	}
}

// Now returns a stubbed time, at epoch.
func Now() metav1.Time {
	return metav1.Time{Time: time.Unix(0, 0)}
}
