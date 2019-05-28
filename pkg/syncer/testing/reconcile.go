package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Token is a test sync token.
const Token = "b38239ea8f58eaed17af6734bd6a025eeafccda1"

var (
	anyContext = gomock.Any()
	anyMessage = gomock.Any()
	anyArgs    = gomock.Any()

	// Converter is an unstructured.Unstructured converter used for testing.
	Converter = runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())

	// ManagementEnabled sets management labels and annotations on the object.
	ManagementEnabled object.Mutator = func(obj *ast.FileObject) {
		object.SetAnnotation(obj.MetaObject(), v1.ResourceManagementKey, v1.ResourceManagementEnabled)
		object.SetLabel(obj.MetaObject(), v1.ManagedByKey, v1.ManagedByValue)
	}
	// ManagementDisabled sets the management disabled annotation on the object.
	ManagementDisabled = object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)
	// ManagementInvalid sets an invalid management annotation on the object.
	ManagementInvalid = object.Annotation(v1.ResourceManagementKey, "invalid")
	// TokenAnnotation sets the sync token annotation on the object
	TokenAnnotation = object.Annotation(v1.SyncTokenAnnotationKey, Token)

	// Herrings is used when the decoder mangles empty vs. non-existent entries in map.
	Herrings = object.Mutate(
		object.Annotation("red", "herring"),
		object.Label("red", "herring"),
	)
)

// NewTestMocks returns a new TestMocks.
func NewTestMocks(t *testing.T, mockCtrl *gomock.Controller) TestMocks {
	return TestMocks{
		t:            t,
		MockCtrl:     mockCtrl,
		MockClient:   mocks.NewMockClient(mockCtrl),
		MockApplier:  mocks.NewMockApplier(mockCtrl),
		MockCache:    mocks.NewMockGenericCache(mockCtrl),
		MockRecorder: mocks.NewMockEventRecorder(mockCtrl),
		MockSignal:   mocks.NewMockRestartSignal(mockCtrl),
	}
}

// TestMocks is a helper used for unit testing controller Reconcile invocation. It wraps all the mocks
// needed to verify common reconcile expectations.
type TestMocks struct {
	t            *testing.T
	MockCtrl     *gomock.Controller
	MockClient   *mocks.MockClient
	MockApplier  *mocks.MockApplier
	MockCache    *mocks.MockGenericCache
	MockRecorder *mocks.MockEventRecorder
	MockSignal   *mocks.MockRestartSignal
}

// ExpectClusterCacheGet stubs the ClusterConfig being fetched from the Cache and verifies we request it.
func (tm *TestMocks) ExpectClusterCacheGet(config *v1.ClusterConfig) {
	if config == nil {
		return
	}
	tm.MockCache.EXPECT().Get(
		anyContext, types.NamespacedName{Name: config.Name}, EqN(tm.t, "ClusterCacheGet", &v1.ClusterConfig{})).
		SetArg(2, *config)
}

// ExpectNamespaceCacheGet organizes the mock calls for first retrieving a NamespaceConfig from the
// cache, and then supplying the Namespace. Correctly returns the intermediate not found error if
// supplied nil for a config.
//
// Does not yet support missing Namespace.
func (tm *TestMocks) ExpectNamespaceCacheGet(config *v1.NamespaceConfig, namespace *corev1.Namespace) {
	if config == nil {
		tm.MockCache.EXPECT().Get(
			anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceConfigCacheGet", &v1.NamespaceConfig{})).
			Return(errors.NewNotFound(schema.GroupResource{}, ""))
	} else {
		tm.MockCache.EXPECT().Get(
			anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceConfigCacheGet",
				&v1.NamespaceConfig{})).
			SetArg(2, *config)
	}
	tm.MockCache.EXPECT().Get(
		anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceCacheGet", &corev1.Namespace{})).
		SetArg(2, *namespace)
}

// ExpectNamespaceUpdate verifies the namespace is updated.
func (tm *TestMocks) ExpectNamespaceUpdate(namespace *corev1.Namespace) {
	if namespace == nil {
		return
	}
	tm.MockClient.EXPECT().Update(
		anyContext, EqN(tm.t, "NamespaceUpdate", namespace))
}

// ExpectNamespaceConfigDelete verifies that the NamespaceConfig is deleted
func (tm *TestMocks) ExpectNamespaceConfigDelete(config *v1.NamespaceConfig) {
	if config == nil {
		return
	}
	tm.MockClient.EXPECT().Delete(anyContext, Eq(tm.t, config), gomock.Any())
}

// ExpectClusterClientGet stubs the ClusterConfig being fetched from the Client and verifies we request it.
func (tm *TestMocks) ExpectClusterClientGet(config *v1.ClusterConfig) {
	if config == nil {
		return
	}
	tm.MockClient.EXPECT().Get(
		anyContext, types.NamespacedName{Name: config.Name}, Eq(tm.t, config))
}

// ExpectNamespaceConfigClientGet stubs the NamespaceConfig being fetched from the Client and verifies we request it.
func (tm *TestMocks) ExpectNamespaceConfigClientGet(config *v1.NamespaceConfig) {
	if config == nil {
		return
	}
	tm.MockClient.EXPECT().Get(
		anyContext, types.NamespacedName{Name: config.Name}, EqN(tm.t, "NamespaceConfigClientGet", config))
}

// ExpectNamespaceClientGet stubs the Namespace being fetched from the Client and verifies we request it.
func (tm *TestMocks) ExpectNamespaceClientGet(namespace *corev1.Namespace) {
	if namespace == nil {
		return
	}
	tm.MockClient.EXPECT().Get(
		anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceClientGet", namespace))
}

// ExpectCacheList stubs the Objects being fetched from the Cache and verifies we request them.
func (tm *TestMocks) ExpectCacheList(gvk schema.GroupVersionKind, namespace string, obj ...runtime.Object) {
	tm.MockCache.EXPECT().
		UnstructuredList(Eq(tm.t, gvk), Eq(tm.t, namespace)).
		Return(ToUnstructuredList(tm.t, Converter, obj...), nil)
}

// ExpectCreate verifies we create the object.
func (tm *TestMocks) ExpectCreate(obj runtime.Object) {
	if obj == nil {
		return
	}
	tm.MockApplier.EXPECT().
		Create(anyContext, Eq(tm.t, ToUnstructured(tm.t, Converter, obj))).
		Return(true, nil)
}

// ExpectUpdate verifies we update the object from original to intended state.
func (tm *TestMocks) ExpectUpdate(d *Diff) {
	if d == nil {
		return
	}
	declared := ToUnstructured(tm.t, Converter, d.Declared)
	actual := ToUnstructured(tm.t, Converter, d.Actual)

	tm.MockApplier.EXPECT().
		Update(anyContext, Eq(tm.t, declared), Eq(tm.t, actual)).
		Return(true, nil)
}

// ExpectDelete verifies we delete the object.
func (tm *TestMocks) ExpectDelete(obj runtime.Object) {
	if obj == nil {
		return
	}
	tm.MockApplier.EXPECT().
		Delete(anyContext, Eq(tm.t, ToUnstructured(tm.t, Converter, obj))).
		Return(true, nil)
}

// ExpectEvent verifies the event was fired.
func (tm *TestMocks) ExpectEvent(event *Event) {
	if event == nil {
		return
	}
	if event.Varargs {
		tm.MockRecorder.EXPECT().
			Eventf(Eq(tm.t, event.Obj), Eq(tm.t, event.Kind), Eq(tm.t, event.Reason), anyMessage, anyArgs)
	} else {
		tm.MockRecorder.EXPECT().
			Event(Eq(tm.t, event.Obj), Eq(tm.t, event.Kind), Eq(tm.t, event.Reason), anyMessage)
	}
}

// ExpectRestart verifies we trigger the Sync controller to restart the SubManager.
func (tm *TestMocks) ExpectRestart(expectRestart bool, source string) {
	if !expectRestart {
		return
	}

	tm.MockSignal.EXPECT().Restart(Eq(tm.t, source))
}

// ExpectClusterStatusUpdate verifies we update the ClusterConfig's status.
func (tm *TestMocks) ExpectClusterStatusUpdate(statusUpdate *v1.ClusterConfig) {
	if statusUpdate == nil {
		return
	}
	mockStatusClient := mocks.NewMockStatusWriter(tm.MockCtrl)
	tm.MockClient.EXPECT().Status().Return(mockStatusClient)
	mockStatusClient.EXPECT().Update(anyContext, Eq(tm.t, statusUpdate))
}

// ExpectNamespaceStatusUpdate verifies we update the NamespaceConfig's status.
func (tm *TestMocks) ExpectNamespaceStatusUpdate(statusUpdate *v1.NamespaceConfig) {
	if statusUpdate == nil {
		return
	}
	mockStatusClient := mocks.NewMockStatusWriter(tm.MockCtrl)
	tm.MockClient.EXPECT().Status().Return(mockStatusClient)
	mockStatusClient.EXPECT().Update(anyContext, Eq(tm.t, statusUpdate))
}

// ToUnstructured converts the object to an unstructured.Unstructured.
func ToUnstructured(t *testing.T, converter runtime.UnstructuredConverter, obj runtime.Object) *unstructured.Unstructured {
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
		result[i] = ToUnstructured(t, converter, obj)
	}
	return result
}

// cmpDiffMatcher returns true iff cmp.Update returns empty string.
// Prints the diff if there is one, as the gomock diff is garbage.
type cmpDiffMatcher struct {
	t        *testing.T
	name     string
	expected interface{}
}

// Eq creates a matcher that compares to expected and prints a diff in case a
// mismatch is found.
func Eq(t *testing.T, expected interface{}) gomock.Matcher {
	return &cmpDiffMatcher{t: t, expected: expected}
}

// EqN creates a matcher that compares to expected and prints a diff in case a
// mismatch is not found.
func EqN(t *testing.T, name string, expected interface{}) gomock.Matcher {
	return &cmpDiffMatcher{t: t, name: name, expected: expected}
}

// String implements Stringer
func (m *cmpDiffMatcher) String() string {
	return fmt.Sprintf("is equal to %v", m.expected)
}

// Matches implements Matcher
func (m *cmpDiffMatcher) Matches(actual interface{}) bool {
	opt := cmpopts.EquateEmpty() // Disregard empty map vs nil.
	if diff := cmp.Diff(m.expected, actual, opt); diff != "" {
		m.t.Logf("The %q matcher has a diff (expected- +actual):%v\n\n", m.name, diff)
		return false
	}
	return true
}

// Diff represents the arguments to an Applier.Updawte invocation.
type Diff struct {
	Declared runtime.Object
	Actual   runtime.Object
}

// Event represents a K8S Event that was emitted as result of the reconcile.
type Event struct {
	// corev1.EventTypeNormal/corev1.EventTypeWarning
	Kind   string
	Reason string
	// set to true if the Event was produced with Eventf (in contrast to Event)
	Varargs bool

	Obj runtime.Object
}

// ImportToken sets the object's import token.
func ImportToken(t string) object.Mutator {
	return func(o *ast.FileObject) {
		switch obj := o.Object.(type) {
		case *v1.ClusterConfig:
			obj.Spec.Token = t
		case *v1.NamespaceConfig:
			obj.Spec.Token = t
		default:
			panic(fmt.Sprintf("Invalid type %T", obj))
		}
	}
}

// SyncTime sets the object's sync time.
func SyncTime() object.Mutator {
	return func(o *ast.FileObject) {
		switch obj := o.Object.(type) {
		case *v1.ClusterConfig:
			obj.Status.SyncTime = Now()
		case *v1.NamespaceConfig:
			obj.Status.SyncTime = Now()
		default:
			panic(fmt.Sprintf("Invalid type %T", obj))
		}
	}
}

// SyncToken sets the object's sync token.
func SyncToken() object.Mutator {
	return func(o *ast.FileObject) {
		switch obj := o.Object.(type) {
		case *v1.ClusterConfig:
			obj.Status.Token = Token
		case *v1.NamespaceConfig:
			obj.Status.Token = Token
		default:
			panic(fmt.Sprintf("Invalid type %T", obj))
		}
	}
}

// MarkForDeletion marks a NamespaceConfig with an intent to be delete
func MarkForDeletion() object.Mutator {
	return func(o *ast.FileObject) {
		switch obj := o.Object.(type) {
		case *v1.NamespaceConfig:
			obj.Spec.DeleteSyncedTime = metav1.Now()
		default:
			panic(fmt.Sprintf("Invalid type %T", obj))
		}
	}
}

// Now returns a stubbed time, at epoch.
func Now() metav1.Time {
	return metav1.Time{Time: time.Unix(0, 0)}
}
