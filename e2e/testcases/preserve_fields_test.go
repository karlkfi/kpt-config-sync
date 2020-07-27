package e2e

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPreserveGeneratedServiceFields(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	// Declare the Service's Namespace
	ns := "autogen-fields"
	nt.Repository.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", ns),
		fake.NamespaceObject(ns))

	// Declare the Service.
	serviceName := "e2e-test-service"
	service := fake.ServiceObject(core.Name(serviceName))
	// The port numbers are arbitrary - just any unused port.
	// Don't reuse these port in other tests just in case.
	targetPort1 := 9376
	targetPort2 := 9377
	service.Spec = corev1.ServiceSpec{
		SessionAffinity: corev1.ServiceAffinityClientIP,
		Selector:        map[string]string{"app": serviceName},
		Type:            corev1.ServiceTypeNodePort,
		Ports: []corev1.ServicePort{{
			Name:       "http",
			Protocol:   corev1.ProtocolTCP,
			Port:       80,
			TargetPort: intstr.FromInt(targetPort1),
		}},
	}
	nt.Repository.Add(fmt.Sprintf("acme/namespaces/%s/service.yaml", ns), service)

	nt.Repository.CommitAndPush("declare Namespace and Service")
	nt.WaitForRepoSync()

	// Ensure the Service has the target port we set.
	err := nt.Validate(serviceName, ns, &corev1.Service{}, hasTargetPort(targetPort1))
	if err != nil {
		t.Fatal(err)
	}

	// We want to wait until the Service specifies ClusterIP and NodePort.
	// We're going to ensure these fields don't change during the test; ACM should
	// not modify these fields since they're never specified and StrategicMergePatch
	// won't overwrite them otherwise.
	var gotService *corev1.Service
	duration, err := nomostest.Retry(60*time.Second, func() error {
		service := &corev1.Service{}
		err := nt.Validate(serviceName, ns, service,
			specifiesClusterIP, specifiesNodePort)
		if err != nil {
			return err
		}
		// The Service specifies the fields we're looking for, so record it.
		gotService = service
		return nil
	})
	t.Logf("waited %v for nodePort and clusterIP to be set", duration)
	if err != nil {
		t.Fatal(err)
	}

	// If strategic merge is NOT being used, Nomos and Kubernetes fight over
	// nodePort.  Nomos constantly deletes the value, and Kubernetes assigns a
	// random value each time. ClusterIP has similar behavior.
	generatedNodePort := gotService.Spec.Ports[0].NodePort
	generatedClusterIP := gotService.Spec.ClusterIP

	// 5 seconds is more than enough time for this to happen.
	_, err = nomostest.Retry(5*time.Second, func() error {
		// This can only return nil if the NodePort/ClusterIP was updated.
		// Potentially flaky check since other things can cause NodePort/ClusterIP
		// to change; copied from bats.
		return nt.Validate(serviceName, ns, &corev1.Service{},
			hasDifferentNodePortOrClusterIP(generatedNodePort, generatedClusterIP))
	})
	if err == nil {
		// We want non-nil error from the Retry above - if err is nil then at least
		// one was incorrectly changed.
		// The node port or cluster IP was updated, so we aren't using StrategicMergePatch.
		t.Fatal("not using strategic merge patch")
	}

	updatedService := service.DeepCopy()
	updatedService.Spec.Ports[0].TargetPort = intstr.FromInt(targetPort2)
	nt.Repository.Add(fmt.Sprintf("acme/namespaces/%s/service.yaml", ns), updatedService)
	nt.Repository.CommitAndPush("update declared Service")
	nt.WaitForRepoSync()

	// Ensure the Service has the new target port we set.
	err = nt.Validate(serviceName, ns, &corev1.Service{}, hasTargetPort(targetPort2))
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreserveGeneratedClusterRoleFields(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	nsViewerName := "namespace-viewer"
	nsViewer := fake.ClusterRoleObject(core.Name(nsViewerName),
		core.Label("permissions", "viewer"))
	nsViewer.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"namespaces"},
		Verbs:     []string{"get", "list"},
	}}
	nt.Repository.Add("acme/cluster/ns-viewer-cr.yaml", nsViewer)

	rbacViewerName := "rbac-viewer"
	rbacViewer := fake.ClusterRoleObject(core.Name(rbacViewerName),
		core.Label("permissions", "viewer"))
	rbacViewer.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{rbacv1.SchemeGroupVersion.Group},
		Resources: []string{"roles", "rolebindings", "clusterroles", "clusterrolebindings"},
		Verbs:     []string{"get", "list"},
	}}
	nt.Repository.Add("acme/cluster/rbac-viewer-cr.yaml", rbacViewer)

	aggregateRoleName := "aggregate"
	aggregateRole := fake.ClusterRoleObject(core.Name(aggregateRoleName))
	aggregateRole.AggregationRule = &rbacv1.AggregationRule{
		ClusterRoleSelectors: []metav1.LabelSelector{{
			MatchLabels: map[string]string{"permissions": "viewer"},
		}},
	}
	nt.Repository.Add("acme/cluster/aggregate-viewer-cr.yaml", aggregateRole)

	nt.Repository.CommitAndPush("declare ClusterRoles")
	nt.WaitForRepoSync()

	// Ensure the aggregate rule is actually aggregated.
	duration, err := nomostest.Retry(60*time.Second, func() error {
		return nt.Validate(aggregateRoleName, "", &rbacv1.ClusterRole{}, hasRules([]rbacv1.PolicyRule{
			nsViewer.Rules[0], rbacViewer.Rules[0],
		}))
	})
	t.Logf("took %v to wait for aggregate ClusterRole", duration)
	if err != nil {
		t.Fatal(err)
	}

	// Update aggregateRole with a new label.
	aggregateRole.Labels["meaningless-label"] = "exists"
	nt.Repository.Add("acme/cluster/aggregate-viewer-cr.yaml", aggregateRole)
	nt.Repository.CommitAndPush("add label to aggregate ClusterRole")
	nt.WaitForRepoSync()

	// Ensure we haven't annihilated the aggregate field.
	err = nt.Validate(aggregateRoleName, "", &rbacv1.ClusterRole{}, hasRules([]rbacv1.PolicyRule{
		nsViewer.Rules[0], rbacViewer.Rules[0],
	}))
	if err != nil {
		t.Error(err)
	}
}

// TestPreserveLastApplied ensures we don't destroy the last-applied-configuration
// annotation.
// TODO(b/160032776): Remove this test once all users are past 1.4.0.
func TestPreserveLastApplied(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	// Declare a ClusterRole and wait for it to sync.
	nsViewerName := "namespace-viewer"
	nsViewer := fake.ClusterRoleObject(core.Name(nsViewerName),
		core.Label("permissions", "viewer"))
	nsViewer.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"namespaces"},
		Verbs:     []string{"get", "list"},
	}}
	nt.Repository.Add("acme/cluster/ns-viewer-cr.yaml", nsViewer)
	nt.Repository.CommitAndPush("add namespace-viewer ClusterRole")
	nt.WaitForRepoSync()

	err := nt.Validate(nsViewerName, "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
	}

	// At this point the fake resource does not have `last-applied-configuration`
	// annotation, so we are setting it rather than overwriting it. The version on
	// cluster has the `last-declared-config` annotation and no
	// `last-applied-annotation`. Since kubectl replace does a delete-then-create,
	// we are effectively recreating the resource with the "last-applied"
	// annotation (which we no longer set starting in 1.4.1). We are then
	// verifying that ConfigSync copies the contents of "last-applied" to
	// "last-declared" and deletes "last-applied".
	nsViewer.Annotations[corev1.LastAppliedConfigAnnotation] = `{"apiVersion":"rbac.authorization.k8s.io/v1","kind":"ClusterRole","metadata":{"annotations":{"configmanagement.gke.io/cluster-name":"e2e-test-cluster","configmanagement.gke.io/managed":"enabled","configmanagement.gke.io/source-path":"cluster/namespace-viewer-clusterrole.yaml"},"labels":{"app.kubernetes.io/managed-by":"configmanagement.gke.io","permissions":"viewer"},"name":"namespace-viewer"},"rules":[{"apiGroups":[""],"resources":["namespaces"],"verbs":["get","list"]}]}`
	nt.Repository.Add("ns-viewer-cr-replace.yaml", nsViewer)
	nt.Kubectl("replace", "-f", filepath.Join(nt.Repository.Root, "ns-viewer-cr-replace.yaml"))

	_, err = nomostest.Retry(20*time.Second, func() error {
		return nt.Validate(nsViewerName, "", &rbacv1.ClusterRole{},
			nomostest.HasExactlyAnnotationKeys(
				corev1.LastAppliedConfigAnnotation,
				v1.ClusterNameAnnotationKey,
				v1.DeclaredConfigAnnotationKey,
				v1.ResourceManagementKey,
				v1.SourcePathAnnotationKey,
				v1.SyncTokenAnnotationKey),
			declaredConfig(nomostest.HasExactlyAnnotationKeys(
				v1.ClusterNameAnnotationKey,
				v1.ResourceManagementKey,
				v1.SourcePathAnnotationKey,
				v1.SyncTokenAnnotationKey)))
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddUpdateDeleteLabels(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	ns := "crud-labels"
	nt.Repository.Add("acme/namespaces/crud-labels/ns.yaml",
		fake.NamespaceObject(ns))

	cmName := "e2e-test-configmap"
	cmPath := "acme/namespaces/crud-labels/configmap.yaml"
	cm := fake.ConfigMapObject(core.Name(cmName))
	nt.Repository.Add(cmPath, cm)
	nt.Repository.CommitAndPush("Adding ConfigMap with no labels to repo")
	nt.WaitForRepoSync()

	// Checking that the configmap with no labels appears on cluster, and
	// that no user labels are specified
	err := nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyLabelKeys(v1.ManagedByKey),
		declaredConfig(nomostest.HasExactlyLabelKeys(v1.ManagedByKey)))
	if err != nil {
		t.Fatal(err)
	}

	cm.Labels["baz"] = "qux"
	nt.Repository.Add(cmPath, cm)
	nt.Repository.CommitAndPush("Update label for ConfigMap in repo")
	nt.WaitForRepoSync()

	// Checking that label is updated after syncing an update.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyLabelKeys(v1.ManagedByKey, "baz"),
		declaredConfig(nomostest.HasExactlyLabelKeys(v1.ManagedByKey, "baz")))
	if err != nil {
		t.Fatal(err)
	}

	delete(cm.Labels, "baz")
	nt.Repository.Add(cmPath, cm)
	nt.Repository.CommitAndPush("Delete label for configmap in repo")
	nt.WaitForRepoSync()

	// Check that the label is deleted after syncing.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyLabelKeys(v1.ManagedByKey),
		declaredConfig(nomostest.HasExactlyLabelKeys(v1.ManagedByKey)))
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddUpdateDeleteAnnotations(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	ns := "crud-annotations"
	nt.Repository.Add("acme/namespaces/crud-annotations/ns.yaml",
		fake.NamespaceObject(ns))

	cmName := "e2e-test-configmap"
	cmPath := "acme/namespaces/crud-annotations/configmap.yaml"
	cm := fake.ConfigMapObject(core.Name(cmName))
	nt.Repository.Add(cmPath, cm)
	nt.Repository.CommitAndPush("Adding ConfigMap with no annotations to repo")
	nt.WaitForRepoSync()

	// Checking that the configmap with no annotations appears on cluster, and
	// that no user annotations are specified
	err := nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyAnnotationKeys(
			v1.ClusterNameAnnotationKey,
			v1.DeclaredConfigAnnotationKey,
			v1.ResourceManagementKey,
			v1.SourcePathAnnotationKey,
			v1.SyncTokenAnnotationKey),
		declaredConfig(nomostest.HasExactlyAnnotationKeys(
			v1.ClusterNameAnnotationKey,
			v1.ResourceManagementKey,
			v1.SourcePathAnnotationKey,
			v1.SyncTokenAnnotationKey)))
	if err != nil {
		t.Fatal(err)
	}

	cm.Annotations["baz"] = "qux"
	nt.Repository.Add(cmPath, cm)
	nt.Repository.CommitAndPush("Update annotation for ConfigMap in repo")
	nt.WaitForRepoSync()

	// Checking that annotation is updated after syncing an update.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyAnnotationKeys("baz",
			v1.ClusterNameAnnotationKey,
			v1.DeclaredConfigAnnotationKey,
			v1.ResourceManagementKey,
			v1.SourcePathAnnotationKey,
			v1.SyncTokenAnnotationKey),
		nomostest.HasAnnotation("baz", "qux"),
		declaredConfig(nomostest.HasExactlyAnnotationKeys("baz",
			v1.ClusterNameAnnotationKey,
			v1.ResourceManagementKey,
			v1.SourcePathAnnotationKey,
			v1.SyncTokenAnnotationKey)),
		nomostest.HasAnnotation("baz", "qux"))
	if err != nil {
		t.Fatal(err)
	}

	delete(cm.Annotations, "baz")
	nt.Repository.Add(cmPath, cm)
	nt.Repository.CommitAndPush("Delete annotation for configmap in repo")
	nt.WaitForRepoSync()

	// Check that the annotation is deleted after syncing.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyAnnotationKeys(
			v1.ClusterNameAnnotationKey,
			v1.DeclaredConfigAnnotationKey,
			v1.ResourceManagementKey,
			v1.SourcePathAnnotationKey,
			v1.SyncTokenAnnotationKey),
		declaredConfig(nomostest.HasExactlyAnnotationKeys(
			v1.ClusterNameAnnotationKey,
			v1.ResourceManagementKey,
			v1.SourcePathAnnotationKey,
			v1.SyncTokenAnnotationKey)))
	if err != nil {
		t.Fatal(err)
	}
}

// declaredConfig wraps a Predicate, applying it to the config specified in the
// declared-config annotation key.
func declaredConfig(p nomostest.Predicate) nomostest.Predicate {
	return func(o core.Object) error {
		declared := o.GetAnnotations()[v1.DeclaredConfigAnnotationKey]
		// For now we don't have any tests that require type-specific structs, so
		// Unstructured is fine.
		u := &unstructured.Unstructured{}
		err := json.Unmarshal([]byte(declared), u)
		if err != nil {
			return err
		}
		return errors.Wrap(p(u), "declared config")
	}
}

func hasDifferentNodePortOrClusterIP(nodePort int32, clusterIP string) nomostest.Predicate {
	// We have to check both in the same Predicate as predicates are AND-ed together.
	// We want to return nil if EITHER nodePort or clusterIP changes.
	return func(o core.Object) error {
		service, ok := o.(*corev1.Service)
		if !ok {
			return nomostest.WrongTypeErr(o, &corev1.Service{})
		}
		gotNodePort := service.Spec.Ports[0].NodePort
		gotClusterIP := service.Spec.ClusterIP
		if nodePort == gotNodePort && clusterIP == gotClusterIP {
			return errors.New("spec.ports[0].nodePort and spec.clusterIP unchanged")
		}
		return nil
	}
}

func specifiesClusterIP(o core.Object) error {
	service, ok := o.(*corev1.Service)
	if !ok {
		return nomostest.WrongTypeErr(o, &corev1.Service{})
	}
	if service.Spec.ClusterIP == "" {
		return errors.New("spec.clusterIP is not set")
	}
	return nil
}

func specifiesNodePort(o core.Object) error {
	service, ok := o.(*corev1.Service)
	if !ok {
		return nomostest.WrongTypeErr(o, &corev1.Service{})
	}
	if service.Spec.Ports[0].NodePort == 0 {
		return errors.New("spec.ports[0].nodePort is not set")
	}
	return nil
}

func hasTargetPort(want int) nomostest.Predicate {
	return func(o core.Object) error {
		service, ok := o.(*corev1.Service)
		if !ok {
			return nomostest.WrongTypeErr(o, &corev1.Service{})
		}
		got := service.Spec.Ports[0].TargetPort.IntValue()
		if want != got {
			return errors.Errorf("port %d synced, want %d", got, want)
		}
		return nil
	}
}
