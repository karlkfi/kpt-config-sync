package quota

import (
	"fmt"
	"log"

	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/cli"
	fakemeta "github.com/google/stolos/pkg/client/meta/fake"
	fakepolicyhierarchy "github.com/google/stolos/pkg/client/policyhierarchy/fake"
	apicore "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"
)

var (
	namespaceTypeMeta = meta.TypeMeta{
		Kind: "Namespace",
	}
	resourceQuotaTypeMeta = meta.TypeMeta{
		Kind: "StolosResourceQuota",
	}
)

func ExampleQuota() {
	tests := []struct {
		namespaces []runtime.Object
		quotas     []runtime.Object
		namespace  string
		err        error
	}{
		{
			// Basic test.
			namespaces: []runtime.Object{
				namespace("default", ""),
			},
			quotas: []runtime.Object{
				stolosResourceQuota("quota1", "default"),
			},
			namespace: "default",
		},
		{
			// Only slightly more complicated.
			namespaces: []runtime.Object{
				namespace("root", ""),
				namespace("ns1", "root"),
				namespace("ns2", "root"),
			},
			quotas: []runtime.Object{
				stolosResourceQuota("quota1", "root"),
				stolosResourceQuota("quota2", "root"),
				stolosResourceQuota("quota3", "ns1"),
				stolosResourceQuota("quota4", "ns2"),
			},
			namespace: "ns1",
		},
	}

	for i, test := range tests {
		ctx := &cli.CommandContext{
			Client:    stolosFakeFromStorage(test.namespaces, test.quotas),
			Namespace: test.namespace,
		}
		fmt.Printf("---\nTest case: %v\n", i)
		err := GetHierarchical(ctx, []string{""})
		if err != nil {
			if test.err == nil {
				log.Printf("[%v] unexpected error: %v", i, err)
			}
		}
	}
	// Output:
	// ---
	// Test case: 0
	// # Namespace: "default"
	// #
	// items:
	// - kind: StolosResourceQuota
	//   metadata:
	//     creationTimestamp: null
	//     name: quota1
	//     namespace: default
	//   spec:
	//     status: {}
	// metadata: {}
	// ---
	// Test case: 1
	// # Namespace: "ns1"
	// #
	// items:
	// - kind: StolosResourceQuota
	//   metadata:
	//     creationTimestamp: null
	//     name: quota3
	//     namespace: ns1
	//   spec:
	//     status: {}
	// metadata: {}
	// # Namespace: "root"
	// #
	// items:
	// - kind: StolosResourceQuota
	//   metadata:
	//     creationTimestamp: null
	//     name: quota1
	//     namespace: root
	//   spec:
	//     status: {}
	// - kind: StolosResourceQuota
	//   metadata:
	//     creationTimestamp: null
	//     name: quota2
	//     namespace: root
	//   spec:
	//     status: {}
	// metadata: {}
}

// TODO(fmil): This should probably be in the policyhierarchy fake.
func stolosFakeFromStorage(
	namespaces []runtime.Object,
	quotas []runtime.Object,
) *fakemeta.Client {
	stolosFake := fakemeta.NewClient()
	// Check whether this works for content.
	stolosFake.KubernetesClientset =
		fakekubernetes.NewSimpleClientset(namespaces...)
	stolosFake.PolicyhierarchyClientset =
		fakepolicyhierarchy.NewSimpleClientset(quotas...)
	return stolosFake
}

// namespace creates a Namespace object named 'name', with
// Stolos-style parent 'parent'.
func namespace(name, parent string) *apicore.Namespace {
	return &apicore.Namespace{
		TypeMeta: namespaceTypeMeta,
		ObjectMeta: meta.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				policyhierarchy.ParentLabelKey: parent,
			},
		},
	}
}

// stolosResourceQuota is similar to above.
func stolosResourceQuota(
	name, namespace string) *policyhierarchy.StolosResourceQuota {
	return &policyhierarchy.StolosResourceQuota{
		TypeMeta: resourceQuotaTypeMeta,
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		// Sic, there is a typo in the quota spec.
		Spec: policyhierarchy.StolosResourceQuotaSpec{},
	}
}
