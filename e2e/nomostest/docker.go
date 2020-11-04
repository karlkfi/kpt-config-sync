package nomostest

import (
	"fmt"
	"time"

	"github.com/google/nomos/e2e/nomostest/docker"
	corev1 "k8s.io/api/core/v1"
)

func connectToLocalRegistry(nt *NT) {
	nt.T.Helper()
	docker.StartLocalRegistry(nt.T)

	// We get access to the kubectl API before the Kind cluster is finished being
	// set up, so the control plane is sometimes still being modified when we do
	// this.
	_, err := Retry(20*time.Second, func() error {
		// See https://kind.sigs.k8s.io/docs/user/local-registry/ for explanation.
		node := &corev1.Node{}
		err := nt.Get(nt.ClusterName+"-control-plane", "", node)
		if err != nil {
			return err
		}
		node.Annotations["kind.x-k8s.io/registry"] = fmt.Sprintf("localhost:%d", docker.RegistryPort)
		return nt.Update(node)
	})
	if err != nil {
		nt.T.Fatalf("connecting cluster to local Docker registry: %v", err)
	}
}
