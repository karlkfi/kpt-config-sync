package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSwitchFromMultiRepoToMonoRepo(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	// Declare the Namespace
	ns := "switch-to-mono"
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", ns),
		fake.NamespaceObject(ns))

	// Declare the Service.
	serviceName := "e2e-test-service"
	service := fake.ServiceObject(core.Name(serviceName))
	// The port numbers are arbitrary - just any unused port.
	// Don't reuse these port in other tests just in case.
	targetPort1 := 9378
	targetPort2 := 9379
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

	var rs v1alpha1.RootSync
	err = nt.Validate(v1alpha1.RootSyncName, v1.NSConfigManagementSystem, &rs)
	if err != nil {
		t.Fatal(err)
	}

	// Delete RootSync custom resource from the cluster.
	err = nt.Delete(&rs)
	if err != nil {
		t.Fatalf("deleting RootSync: %v", err)
	}

	// Verify Root Reconciler deployment no longer present.
	_, err = nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(reconciler.RootSyncName, v1.NSConfigManagementSystem, fake.DeploymentObject())
	})
	if err != nil {
		t.Errorf("Reconciler deployment present after deletion: %v", err)
	}

	// Switch to mono-repo mode.
	nomostest.SwitchMode(t, nt)

	nt.WaitForRepoSyncs()
	// Ensure the Service exists and has the target port we set.
	err = nt.Validate(serviceName, ns, &corev1.Service{}, hasTargetPort(targetPort1))
	if err != nil {
		t.Fatal(err)
	}

	updatedService := service.DeepCopy()
	updatedService.Spec.Ports[0].TargetPort = intstr.FromInt(targetPort2)
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/service.yaml", ns), updatedService)
	nt.Root.CommitAndPush("update declared Service")
	nt.WaitForRepoSyncs()

	// Ensure the Service exists and has the target port we set.
	err = nt.Validate(serviceName, ns, &corev1.Service{}, hasTargetPort(targetPort2))
	if err != nil {
		t.Fatal(err)
	}
}

func TestSwitchFromMonoRepoToMultiRepo(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMultiRepo)

	// Declare the Namespace
	ns := "switch-to-csmr"
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", ns),
		fake.NamespaceObject(ns))

	// Declare the Service.
	serviceName := "e2e-test-service"
	service := fake.ServiceObject(core.Name(serviceName))
	// The port numbers are arbitrary - just any unused port.
	// Don't reuse these port in other tests just in case.
	targetPort1 := 9380
	targetPort2 := 9381
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

	d := fake.DeploymentObject()
	err = nt.Validate(filesystem.GitImporterName, v1.NSConfigManagementSystem, d)
	if err != nil {
		t.Fatal(err)
	}

	// Delete git-importer from the cluster.
	err = nt.Delete(d)
	if err != nil {
		t.Fatalf("deleting Repo: %v", err)
	}

	// Verify git-importer no longer present.
	_, err = nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(filesystem.GitImporterName, v1.NSConfigManagementSystem, fake.DeploymentObject())
	})
	if err != nil {
		t.Errorf("Git importer deployment present after deletion: %v", err)
	}

	// Switch to multi-repo mode.
	nomostest.SwitchMode(t, nt)

	nt.WaitForRootSync(kinds.RootSync(),
		"root-sync", configmanagement.ControllerNamespace, nomostest.RootSyncHasStatusSyncCommit)
	// Ensure the Service exists and has the target port we set.
	err = nt.Validate(serviceName, ns, &corev1.Service{}, hasTargetPort(targetPort1))
	if err != nil {
		t.Fatal(err)
	}

	updatedService := service.DeepCopy()
	updatedService.Spec.Ports[0].TargetPort = intstr.FromInt(targetPort2)
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/service.yaml", ns), updatedService)
	nt.Root.CommitAndPush("update declared Service")
	nt.WaitForRootSync(kinds.RootSync(),
		"root-sync", configmanagement.ControllerNamespace, nomostest.RootSyncHasStatusSyncCommit)

	// Ensure the Service exists and has the target port we set.
	err = nt.Validate(serviceName, ns, &corev1.Service{}, hasTargetPort(targetPort2))
	if err != nil {
		t.Fatal(err)
	}
}
