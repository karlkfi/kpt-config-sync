// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	// The GCENode* values are interpolated in the prepareGCENodeSnippet function
	// Keep the image tag consistent with https://team.git.corp.google.com/nomos-team/nomos-operator/+/refs/heads/master/pkg/controller/configsync/constants.go#34.
	gceNodeAskpassImageTag = "20210831174857"
	// GceNodeAskpassSidecarName is the container name of gcenode-askpass-sidecar.
	GceNodeAskpassSidecarName = "gcenode-askpass-sidecar"
	gceNodeAskpassPort        = 9102
)

func gceNodeAskPassContainerImage(name, tag string) string {
	return fmt.Sprintf("gcr.io/config-management-release/%v:%v", name, tag)
}

func configureGceNodeAskPass(cr *corev1.Container) {
	cr.Name = GceNodeAskpassSidecarName
	cr.Image = gceNodeAskPassContainerImage(GceNodeAskpassSidecarName, gceNodeAskpassImageTag)
	cr.Args = addPort(gceNodeAskpassPort)
	cr.SecurityContext = setSecurityContext()
	cr.TerminationMessagePolicy = corev1.TerminationMessageReadFile
	cr.TerminationMessagePath = corev1.TerminationMessagePathDefault
	cr.ImagePullPolicy = corev1.PullIfNotPresent
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
		if c.Name == GceNodeAskpassSidecarName {
			return true
		}
	}
	return false
}
