package controllers

import (
	"sort"

	"github.com/google/nomos/pkg/reconcilermanager/controllers/secrets"
	corev1 "k8s.io/api/core/v1"
)

// GitCredentialVolume is the volume name of the git credentials.
const GitCredentialVolume = "git-creds"

func filterVolumes(existing []corev1.Volume, authType string, secretName string) []corev1.Volume {
	var updatedVolumes []corev1.Volume

	for _, volume := range existing {
		if volume.Name == GitCredentialVolume {
			// Don't mount git-creds volume if auth is 'none' or 'gcenode'
			if secrets.SkipForAuth(authType) {
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
		if secrets.SkipForAuth(auth) && volume.Name == GitCredentialVolume {
			continue
		}
		volumeMount = append(volumeMount, volume)
	}
	sort.Slice(volumeMount[:], func(i, j int) bool {
		return volumeMount[i].Name < volumeMount[j].Name
	})
	return volumeMount
}
