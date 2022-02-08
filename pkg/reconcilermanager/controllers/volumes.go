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
	"sort"

	corev1 "k8s.io/api/core/v1"
)

// GitCredentialVolume is the volume name of the git credentials.
const GitCredentialVolume = "git-creds"

func filterVolumes(existing []corev1.Volume, authType string, secretName string) []corev1.Volume {
	var updatedVolumes []corev1.Volume

	for _, volume := range existing {
		if volume.Name == GitCredentialVolume {
			// Don't mount git-creds volume if auth is 'none' or 'gcenode'
			if SkipForAuth(authType) {
				continue
			}
			volume.Secret.SecretName = secretName
		}
		updatedVolumes = append(updatedVolumes, volume)
	}

	return updatedVolumes
}

// volumeMounts returns a sorted list of VolumeMounts by filtering out git-creds
// VolumeMount when secret is 'none' or 'gcenode'.
func volumeMounts(auth string, vm []corev1.VolumeMount) []corev1.VolumeMount {
	var volumeMount []corev1.VolumeMount
	for _, volume := range vm {
		if SkipForAuth(auth) && volume.Name == GitCredentialVolume {
			continue
		}
		volumeMount = append(volumeMount, volume)
	}
	sort.Slice(volumeMount[:], func(i, j int) bool {
		return volumeMount[i].Name < volumeMount[j].Name
	})
	return volumeMount
}
