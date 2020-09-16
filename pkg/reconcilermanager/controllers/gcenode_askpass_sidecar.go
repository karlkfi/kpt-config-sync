package controllers

import (
	"fmt"

	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func gceNodeAskPassSidecar() corev1.Container {
	return corev1.Container{
		Name:  gceNodeAskpassSidecarName,
		Image: fmt.Sprintf("gcr.io/config-management-release/%v:%v", gceNodeAskpassSidecarName, gceNodeAskpassImageTag),
		Args:  addPort(gceNodeAskpassPort),
	}
}

func authTypeGCENode(secret string) bool {
	return v1alpha1.GitSecretGCENode == secret
}

func addPort(port int) []string {
	return []string{fmt.Sprintf("--port=%v", port)}
}
