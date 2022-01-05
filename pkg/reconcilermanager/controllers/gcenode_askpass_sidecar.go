package controllers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	// The GCENode* values are interpolated in the prepareGCENodeSnippet function
	// Keep the image tag consistent with https://team.git.corp.google.com/nomos-team/nomos-operator/+/refs/heads/master/pkg/controller/configsync/constants.go#34.
	gceNodeAskpassImageTag    = "20210831174857"
	gceNodeAskpassSidecarName = "gcenode-askpass-sidecar"
	gceNodeAskpassPort        = 9102
)

func gceNodeAskPassContainerImage(name, tag string) string {
	return fmt.Sprintf("gcr.io/config-management-release/%v:%v", name, tag)
}

func configureGceNodeAskPass(cr *corev1.Container) {
	cr.Name = gceNodeAskpassSidecarName
	cr.Image = gceNodeAskPassContainerImage(gceNodeAskpassSidecarName, gceNodeAskpassImageTag)
	cr.Args = addPort(gceNodeAskpassPort)
	cr.SecurityContext = setSecurityContext()
}

func gceNodeAskPassSidecar() corev1.Container {
	var cr corev1.Container
	configureGceNodeAskPass(&cr)
	return cr
}

func addPort(port int) []string {
	return []string{fmt.Sprintf("--port=%v", port)}
}

// setSecurityContext sets the security context for the gcenode-askpass-sidecar container.
// It drops the NET_RAW capability, disallows privilege escalation and read-only root filesystem.
func setSecurityContext() *corev1.SecurityContext {
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := false
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"NET_RAW"},
		},
	}
}

// containsGCENodeAskPassSidecar checks whether gcenode-askpass-sidecar is
// already present in templateSpec.Containers
func containsGCENodeAskPassSidecar(cs []corev1.Container) bool {
	for _, c := range cs {
		if c.Name == gceNodeAskpassSidecarName {
			return true
		}
	}
	return false
}
