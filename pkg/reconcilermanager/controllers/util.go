package controllers

import (
	"strings"

	"github.com/google/nomos/pkg/importer/filesystem"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func gitSyncData(branch, repo string) map[string]string {
	result := make(map[string]string)
	result["GIT_KNOWN_HOSTS"] = "false"
	result["GIT_SYNC_BRANCH"] = branch
	result["GIT_SYNC_REPO"] = repo
	result["GIT_SYNC_REV"] = "HEAD"
	result["GIT_SYNC_WAIT"] = "15"
	return result
}

// reconcilerData returns configmap data for namespace reconciler.
func reconcilerData(reconcilerScope, policyDir string) map[string]string {
	result := make(map[string]string)
	result["RECONCILER_SCOPE"] = reconcilerScope
	result["POLICY_DIR"] = policyDir
	return result
}

// rootReconcilerData returns configmap data for root reconciler.
func rootReconcilerData(reconcilerScope, policyDir, clusterName string) map[string]string {
	result := reconcilerData(reconcilerScope, policyDir)
	result["CLUSTER_NAME"] = clusterName
	return result
}

func sourceFormatData(format string) map[string]string {
	result := make(map[string]string)
	result[filesystem.SourceFormatKey] = format
	return result
}

func ownerReference(kind, name string, uid types.UID) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		{
			APIVersion:         v1.SchemeGroupVersion.String(),
			Kind:               kind,
			Name:               name,
			Controller:         pointer.BoolPtr(true),
			BlockOwnerDeletion: pointer.BoolPtr(true),
			UID:                uid,
		},
	}
}

func envFromSources(configmapRef map[string]*bool) []corev1.EnvFromSource {
	var envFromSource []corev1.EnvFromSource
	for name, optional := range configmapRef {
		cfgMap := corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Optional: optional,
			},
		}
		envFromSource = append(envFromSource, cfgMap)
	}
	return envFromSource
}

func buildRepoSyncName(names ...string) string {
	prefix := []string{repoSyncReconcilerPrefix}
	return strings.Join(append(prefix, names...), "-")
}

func buildRootSyncName(names ...string) string {
	prefix := []string{rootSyncReconcilerName}
	return strings.Join(append(prefix, names...), "-")
}
