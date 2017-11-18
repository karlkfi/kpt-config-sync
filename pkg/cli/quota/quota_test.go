package quota

import (
	"fmt"
	"log"

	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/cli"
	"github.com/google/stolos/pkg/cli/testing"
	fakemeta "github.com/google/stolos/pkg/client/meta/fake"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
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
				testing.NewNamespace("default", ""),
			},
			quotas: []runtime.Object{
				stolosResourceQuota("quota1", "default"),
			},
			namespace: "default",
		},
		{
			// Only slightly more complicated.
			namespaces: []runtime.Object{
				testing.NewNamespace("root", ""),
				testing.NewNamespace("ns1", "root"),
				testing.NewNamespace("ns2", "root"),
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
			Client:    fakemeta.NewClientWithStorage(test.namespaces, test.quotas),
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

// stolosResourceQuota creates a new dummy stolos resource quota with given name
// and scoped to the given namespace.
func stolosResourceQuota(
	name, namespace string) *policyhierarchy.StolosResourceQuota {
	return &policyhierarchy.StolosResourceQuota{
		TypeMeta: resourceQuotaTypeMeta,
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: policyhierarchy.StolosResourceQuotaSpec{},
	}
}
