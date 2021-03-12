package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestAdmission(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	nt.Root.Add("acme/namespaces/hello/ns.yaml",
		fake.NamespaceObject("hello",
			core.Annotation("goodbye", "moon")))
	nt.Root.CommitAndPush("add Namespace")
	nt.WaitForRepoSyncs()

	// Ensure we properly forbid changing declared information.

	// Prevent deleting declared objects.
	_, err := nt.Kubectl("delete", "ns", "hello")
	if err == nil {
		t.Fatal("got `kubectl delete ns hello` success, want return err")
	}

	// Prevent changing declared data.
	_, err = nt.Kubectl("annotate", "--overwrite", "ns", "hello", "goodbye=world")
	if err == nil {
		t.Fatal("got `kubectl annotate --overwrite ns hello goodbye=world` success, want return err")
	}

	// Prevent removing declared data from declared objects.
	_, err = nt.Kubectl("annotate", "ns", "hello", "goodbye-")
	if err == nil {
		t.Fatal("got `kubectl annotate ns hello goodbye-` success, want return err")
	}

	// Ensure we allow changing information which is not declared.

	// Allow adding data in declared objects.
	out, err := nt.Kubectl("annotate", "ns", "hello", "stop=go")
	if err != nil {
		t.Fatalf("got `kubectl annotate ns hello stop=go` error %v %s, want return nil", err, out)
	}

	// Allow changing non-declared data in declared objects.
	out, err = nt.Kubectl("annotate", "--overwrite", "ns", "hello", "stop='oh no'")
	if err != nil {
		t.Fatalf("got `kubectl annotate --overwrite ns hello stop='oh no'` error %v %s, want return nil", err, out)
	}

	// Allow reing non-declared data in declared objects.
	out, err = nt.Kubectl("annotate", "ns", "hello", "stop-")
	if err != nil {
		t.Fatalf("got `kubectl annotate ns hello stop-` error %v %s, want return nil", err, out)
	}
}

func TestIgnoreMutations(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	nt.Root.Add("acme/namespaces/hello/ns.yaml",
		fake.NamespaceObject("hello",
			core.Annotation("goodbye", "moon"),
			core.Annotation(v1beta1.LifecycleMutationAnnotation, v1beta1.IgnoreMutation)))
	nt.Root.CommitAndPush("add Namespace")
	nt.WaitForRepoSyncs()

	// Ensure we properly forbid changing declared information.

	// Prevent deleting declared objects.
	_, err := nt.Kubectl("delete", "ns", "hello")
	if err == nil {
		t.Fatal("got `kubectl delete ns hello` success, want return err")
	}

	// Allow changing declared data.
	out, err := nt.Kubectl("annotate", "--overwrite", "ns", "hello", "goodbye=world")
	if err != nil {
		t.Fatalf("got `kubectl annotate --overwrite ns hello goodbye=world` error %v %s, want success", err, out)
	}
	err = nt.Validate("hello", "", &corev1.Namespace{},
		nomostest.HasAnnotation("goodbye", "world"))
	if err != nil {
		t.Fatalf("got annotation 'goodbye' != 'world', want == 'world'")
	}

	// Allow removing declared data from declared objects.
	out, err = nt.Kubectl("annotate", "ns", "hello", "goodbye-")
	if err != nil {
		t.Fatalf("got `kubectl annotate ns hello goodbye-` error %v %s, want success", err, out)
	}
	err = nt.Validate("hello", "", &corev1.Namespace{},
		nomostest.MissingAnnotation("goodbye"))
	if err != nil {
		t.Fatalf("got annotation 'goodbye' present, want missing")
	}
}
