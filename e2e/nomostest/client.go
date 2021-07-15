package nomostest

import (
	"strings"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/e2e/nomostest/testing"
	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	configsyncv1alpha1 "github.com/google/nomos/pkg/api/configsync/v1alpha1"
	configsyncv1beta1 "github.com/google/nomos/pkg/api/configsync/v1beta1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// connect creates a client.Client to the cluster
func connect(t testing.NTB, cfg *rest.Config, scheme *runtime.Scheme) client.Client {
	t.Helper()

	t.Log("creating Client")
	c, err := client.New(cfg, client.Options{
		// The Scheme is client-side, but this automatically fetches the RestMapper
		// from the cluster.
		Scheme: scheme,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// newScheme creates a new scheme to use to map Go types to types on a
// Kubernetes cluster.
func newScheme(t testing.NTB) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	// It is always safe to add new schemes so long as there are no GVK or struct
	// collisions.
	//
	// We have no tests which require configuring this in incompatible ways, so if
	// you need new types then add them here.
	builders := []runtime.SchemeBuilder{
		admissionv1.SchemeBuilder,
		apiextensionsv1beta1.SchemeBuilder,
		apiextensionsv1.SchemeBuilder,
		appsv1.SchemeBuilder,
		corev1.SchemeBuilder,
		configmanagementv1.SchemeBuilder,
		configsyncv1alpha1.SchemeBuilder,
		configsyncv1beta1.SchemeBuilder,
		policyv1beta1.SchemeBuilder,
		rbacv1.SchemeBuilder,
		rbacv1beta1.SchemeBuilder,
	}
	for _, b := range builders {
		err := b.AddToScheme(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	return s
}

// RestConfig sets up the config for creating a Client connection to a K8s cluster.
// If --test-cluster=kind, it creates a Kind cluster.
// If --test-cluster=kubeconfig, it uses the context specified in kubeconfig.
func RestConfig(t testing.NTB, optsStruct *ntopts.New) {
	switch strings.ToLower(*e2e.TestCluster) {
	case e2e.Kind:
		ntopts.Kind(t, *e2e.KubernetesVersion)(optsStruct)
	case e2e.GKE:
		ntopts.GKECluster(t)(optsStruct)
	default:
		t.Fatalf("unsupported test cluster config %s. Allowed values are %s and %s.", *e2e.TestCluster, e2e.GKE, e2e.Kind)
	}
}
