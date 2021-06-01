package nomostest

import (
	"fmt"
	"time"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	monoRepoCRDs = []string{
		"clusterconfigs.configmanagement.gke.io",
		"hierarchyconfigs.configmanagement.gke.io",
		"namespaceconfigs.configmanagement.gke.io",
		"repos.configmanagement.gke.io",
		"syncs.configmanagement.gke.io",
		// Shared CRDS
		"clusters.clusterregistry.k8s.io",
		"clusterselectors.configmanagement.gke.io",
		"namespaceselectors.configmanagement.gke.io",
	}
	multiRepoCRDs = []string{
		"reposyncs.configsync.gke.io",
		"rootsyncs.configsync.gke.io",
		"resourcegroups.kpt.dev",
		// Shared CRDS
		"clusters.clusterregistry.k8s.io",
		"clusterselectors.configmanagement.gke.io",
		"namespaceselectors.configmanagement.gke.io",
	}
)

// waitForCRDs waits until the specified CRDs are established on the cluster.
func waitForCRDs(nt *NT, crds []string) error {
	took, err := Retry(60*time.Second, func() error {
		for _, crd := range crds {
			err := nt.Validate(crd, "", &v1.CustomResourceDefinition{}, IsEstablished)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	nt.T.Logf("took %v to wait for CRDs: %v", took, crds)
	return nil
}

// IsEstablished returns true if the given CRD is established on the cluster,
// which indicates if discovery knows about it yet. For more info see
// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#create-a-customresourcedefinition
func IsEstablished(o client.Object) error {
	crd, ok := o.(*v1.CustomResourceDefinition)
	if !ok {
		return WrongTypeErr(o, crd)
	}

	for _, condition := range crd.Status.Conditions {
		if condition.Type == v1.Established {
			if condition.Status == v1.ConditionTrue {
				return nil
			}
		}
	}
	return fmt.Errorf("CRD %q is not established", crd.Name)
}
