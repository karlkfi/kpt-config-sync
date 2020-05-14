package nomostest

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// connect creates a client.Client to the cluster
func connect(t *testing.T, cfg *rest.Config) client.Client {
	t.Log("creating Client")
	c, err := client.New(cfg, client.Options{
		// The Scheme is client-side, but this automatically fetches the RestMapper
		// from the cluster.
		Scheme: newScheme(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// newScheme creates a new scheme to use to map Go types to types on a
// Kubernetes cluster.
func newScheme(t *testing.T) *runtime.Scheme {
	s := runtime.NewScheme()

	// It is always safe to add new schemes so long as there are no GVK or struct
	// collisions.
	//
	// We have no tests which require configuring this in incompatible ways, so if
	// you need new types then add them here.
	builders := []runtime.SchemeBuilder{
		corev1.SchemeBuilder,
	}
	for _, b := range builders {
		err := b.AddToScheme(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	return s
}
