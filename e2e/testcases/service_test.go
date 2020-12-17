package e2e

import (
	"fmt"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	corev1 "k8s.io/api/core/v1"
)

func TestMetricsServices(t *testing.T) {
	ns := "foo"
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(ns))

	err := nt.Validate("root-reconciler", configmanagement.ControllerNamespace, &corev1.Service{})
	if err != nil {
		t.Error(err)
	}

	err = nt.Validate(fmt.Sprintf("ns-reconciler-%s", ns), configmanagement.ControllerNamespace, &corev1.Service{})
	if err != nil {
		t.Error(err)
	}
}
