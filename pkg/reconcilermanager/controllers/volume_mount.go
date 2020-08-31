package controllers

import (
	"sort"

	"github.com/google/nomos/pkg/reconcilermanager/controllers/secret"
	corev1 "k8s.io/api/core/v1"
)

// volumeMounts returns a sorted list of volumemounts by filtering out git-creds
// volumemount when secret is 'none' or 'gcenode'.
func volumeMounts(auth string, vm []corev1.VolumeMount) []corev1.VolumeMount {
	var volumeMount []corev1.VolumeMount
	for _, volume := range vm {
		if secret.SkipForAuth(auth) && volume.Name == gitCredentialVolume {
			continue
		}
		volumeMount = append(volumeMount, volume)
	}
	sort.Slice(volumeMount[:], func(i, j int) bool {
		return volumeMount[i].Name < volumeMount[j].Name
	})
	return volumeMount
}
