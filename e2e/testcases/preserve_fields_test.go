package e2e

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/webhook/configuration"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPreserveGeneratedServiceFields(t *testing.T) {
	nt := nomostest.New(t)

	// Declare the Service's Namespace
	ns := "autogen-fields"
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", ns),
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
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/service.yaml", ns), service)

	nt.Root.CommitAndPush("declare Namespace and Service")
	nt.WaitForRepoSyncs()

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

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("Service"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	updatedService := service.DeepCopy()
	updatedService.Spec.Ports[0].TargetPort = intstr.FromInt(targetPort2)
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/service.yaml", ns), updatedService)
	nt.Root.CommitAndPush("update declared Service")
	nt.WaitForRepoSyncs()

	// Ensure the Service has the new target port we set.
	err = nt.Validate(serviceName, ns, &corev1.Service{}, hasTargetPort(targetPort2))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourcePatched("Namespace", 2), metrics.ResourcePatched("Service", 2))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}
}

func TestPreserveGeneratedClusterRoleFields(t *testing.T) {
	nt := nomostest.New(t)

	nsViewerName := "namespace-viewer"
	nsViewer := fake.ClusterRoleObject(core.Name(nsViewerName),
		core.Label("permissions", "viewer"))
	nsViewer.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"namespaces"},
		Verbs:     []string{"get", "list"},
	}}
	nt.Root.Add("acme/cluster/ns-viewer-cr.yaml", nsViewer)

	rbacViewerName := "rbac-viewer"
	rbacViewer := fake.ClusterRoleObject(core.Name(rbacViewerName),
		core.Label("permissions", "viewer"))
	rbacViewer.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{rbacv1.SchemeGroupVersion.Group},
		Resources: []string{"roles", "rolebindings", "clusterroles", "clusterrolebindings"},
		Verbs:     []string{"get", "list"},
	}}
	nt.Root.Add("acme/cluster/rbac-viewer-cr.yaml", rbacViewer)

	aggregateRoleName := "aggregate"
	// We have to declare the YAML explicitly because otherwise the declaration
	// explicitly declares "rules: []" due to how Go handles empty/unset fields.
	nt.Root.AddFile("acme/cluster/aggregate-viewer-cr.yaml", []byte(`
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: aggregate
aggregationRule:
  clusterRoleSelectors:
  - matchLabels:
      permissions: viewer`))

	nt.Root.CommitAndPush("declare ClusterRoles")
	nt.WaitForRepoSyncs()

	// Ensure the aggregate rule is actually aggregated.
	duration, err := nomostest.Retry(20*time.Second, func() error {
		return nt.Validate(aggregateRoleName, "", &rbacv1.ClusterRole{}, clusterRoleHasRules([]rbacv1.PolicyRule{
			nsViewer.Rules[0], rbacViewer.Rules[0],
		}))
	})
	t.Logf("took %v to wait for aggregate ClusterRole", duration)
	if err != nil {
		t.Fatal(err)
	}

	// Update aggregateRole with a new label.
	nt.Root.AddFile("acme/cluster/aggregate-viewer-cr.yaml", []byte(`
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: aggregate
  labels:
    meaningless-label: exists
aggregationRule:
  clusterRoleSelectors:
  - matchLabels:
      permissions: viewer`))
	nt.Root.CommitAndPush("add label to aggregate ClusterRole")
	nt.WaitForRepoSyncs()

	// Ensure we don't overwrite the aggregate rules.
	err = nt.Validate(aggregateRoleName, "", &rbacv1.ClusterRole{},
		clusterRoleHasRules([]rbacv1.PolicyRule{
			nsViewer.Rules[0], rbacViewer.Rules[0],
		}))
	if err != nil {
		t.Fatal(err)
	}

	// Validate no error metrics are emitted.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating error metrics: %v", err)
	}
}

// TestPreserveLastApplied ensures we don't destroy the last-applied-configuration
// annotation.
// TODO(b/160032776): Remove this test once all users are past 1.4.0.
func TestPreserveLastApplied(t *testing.T) {
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
	nt.Root.Add("acme/cluster/ns-viewer-cr.yaml", nsViewer)
	nt.Root.CommitAndPush("add namespace-viewer ClusterRole")
	nt.WaitForRepoSyncs()

	err := nt.Validate(nsViewerName, "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
	}

	annotationKeys := []string{
		v1.ClusterNameAnnotationKey,
		v1.ResourceManagementKey,
		v1.SourcePathAnnotationKey,
		v1.SyncTokenAnnotationKey,
		v1alpha1.DeclaredFieldsKey,
	}
	if nt.MultiRepo {
		annotationKeys = append(annotationKeys, v1alpha1.GitContextKey, v1alpha1.ResourceManagerKey, kptapplier.OwningInventoryKey)
	}
	withDeclared := append([]string{corev1.LastAppliedConfigAnnotation}, annotationKeys...)

	nsViewer.Annotations[corev1.LastAppliedConfigAnnotation] = `{"apiVersion":"rbac.authorization.k8s.io/v1","kind":"ClusterRole","metadata":{"annotations":{"configmanagement.gke.io/cluster-name":"e2e-test-cluster","configmanagement.gke.io/managed":"enabled","configmanagement.gke.io/source-path":"cluster/namespace-viewer-clusterrole.yaml"},"labels":{"app.kubernetes.io/managed-by":"configmanagement.gke.io","permissions":"viewer"},"name":"namespace-viewer"},"rules":[{"apiGroups":[""],"resources":["namespaces"],"verbs":["get","list"]}]}`
	nt.Root.Add("ns-viewer-cr-replace.yaml", nsViewer)
	if nt.MultiRepo {
		// Admission webhook denies change. We don't get a "LastApplied" annotation
		// as we prevented the change outright.
		_, err = nt.Kubectl("replace", "-f", filepath.Join(nt.Root.Root, "ns-viewer-cr-replace.yaml"))
		if err == nil {
			t.Fatal("got kubectl replace err = nil, want admission webhook to deny")
		}

		_, err = nomostest.Retry(20*time.Second, func() error {
			return nt.Validate(nsViewerName, "", &rbacv1.ClusterRole{},
				nomostest.HasExactlyAnnotationKeys(annotationKeys...))
		})
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// No admission webhook in mono repo.

		// At this point the fake resource does not have `last-applied-configuration`
		// annotation, so we are setting it rather than overwriting it. The version on
		// cluster has the `last-declared-config` annotation and no
		// `last-applied-annotation`. Since kubectl replace does a delete-then-create,
		// we are effectively recreating the resource with the "last-applied"
		// annotation (which we no longer set starting in 1.4.1). We are then
		// verifying that ConfigSync copies the contents of "last-applied" to
		// "last-declared" and deletes "last-applied".
		nt.MustKubectl("replace", "-f", filepath.Join(nt.Root.Root, "ns-viewer-cr-replace.yaml"))

		_, err = nomostest.Retry(20*time.Second, func() error {
			return nt.Validate(nsViewerName, "", &rbacv1.ClusterRole{},
				nomostest.HasExactlyAnnotationKeys(withDeclared...))
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Validate no error metrics are emitted.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating error metrics: %v", err)
	}
}

func TestAddUpdateDeleteLabels(t *testing.T) {
	nt := nomostest.New(t)

	ns := "crud-labels"
	nt.Root.Add("acme/namespaces/crud-labels/ns.yaml",
		fake.NamespaceObject(ns))

	cmName := "e2e-test-configmap"
	cmPath := "acme/namespaces/crud-labels/configmap.yaml"
	cm := fake.ConfigMapObject(core.Name(cmName))
	nt.Root.Add(cmPath, cm)
	nt.Root.CommitAndPush("Adding ConfigMap with no labels to repo")
	nt.WaitForRepoSyncs()

	var defaultLabels = []string{v1.ManagedByKey, configuration.DeclaredVersionLabel}

	// Checking that the configmap with no labels appears on cluster, and
	// that no user labels are specified
	err := nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyLabelKeys(defaultLabels...))
	if err != nil {
		t.Fatal(err)
	}

	cm.Labels["baz"] = "qux"
	nt.Root.Add(cmPath, cm)
	nt.Root.CommitAndPush("Update label for ConfigMap in repo")
	nt.WaitForRepoSyncs()

	// Checking that label is updated after syncing an update.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyLabelKeys(append(defaultLabels, "baz")...))
	if err != nil {
		t.Fatal(err)
	}

	delete(cm.Labels, "baz")
	nt.Root.Add(cmPath, cm)
	nt.Root.CommitAndPush("Delete label for configmap in repo")
	nt.WaitForRepoSyncs()

	// Check that the label is deleted after syncing.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyLabelKeys(v1.ManagedByKey, configuration.DeclaredVersionLabel))
	if err != nil {
		t.Fatal(err)
	}

	// Validate no error metrics are emitted.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating error metrics: %v", err)
	}
}

func TestAddUpdateDeleteAnnotations(t *testing.T) {
	nt := nomostest.New(t)

	ns := "crud-annotations"
	nt.Root.Add("acme/namespaces/crud-annotations/ns.yaml",
		fake.NamespaceObject(ns))

	cmName := "e2e-test-configmap"
	cmPath := "acme/namespaces/crud-annotations/configmap.yaml"
	cm := fake.ConfigMapObject(core.Name(cmName))
	nt.Root.Add(cmPath, cm)
	nt.Root.CommitAndPush("Adding ConfigMap with no annotations to repo")
	nt.WaitForRepoSyncs()

	annotationKeys := []string{
		v1.ClusterNameAnnotationKey,
		v1.ResourceManagementKey,
		v1.SourcePathAnnotationKey,
		v1.SyncTokenAnnotationKey,
		v1alpha1.DeclaredFieldsKey,
	}
	if nt.MultiRepo {
		annotationKeys = append(annotationKeys, v1alpha1.GitContextKey, v1alpha1.ResourceManagerKey, kptapplier.OwningInventoryKey)
	}

	// Checking that the configmap with no annotations appears on cluster, and
	// that no user annotations are specified
	err := nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyAnnotationKeys(annotationKeys...))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("ConfigMap"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	cm.Annotations["baz"] = "qux"
	nt.Root.Add(cmPath, cm)
	nt.Root.CommitAndPush("Update annotation for ConfigMap in repo")
	nt.WaitForRepoSyncs()

	updatedKeys := append([]string{"baz"}, annotationKeys...)

	// Checking that annotation is updated after syncing an update.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyAnnotationKeys(updatedKeys...),
		nomostest.HasAnnotation("baz", "qux"))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourcePatched("Namespace", 2), metrics.ResourcePatched("ConfigMap", 2))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	delete(cm.Annotations, "baz")
	nt.Root.Add(cmPath, cm)
	nt.Root.CommitAndPush("Delete annotation for configmap in repo")
	nt.WaitForRepoSyncs()

	// Check that the annotation is deleted after syncing.
	err = nt.Validate(cmName, ns, &corev1.ConfigMap{},
		nomostest.HasExactlyAnnotationKeys(annotationKeys...))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourcePatched("Namespace", 3), metrics.ResourcePatched("ConfigMap", 3))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
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
