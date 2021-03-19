package controllers

import (
	"sort"

	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/reconcilermanager"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// reconcilerData returns configmap data for namespace reconciler.
func reconcilerData(clusterName string, reconcilerScope declared.Scope, gitConfig *v1alpha1.Git, pollPeriod string) map[string]string {
	result := make(map[string]string)
	result[reconcilermanager.ClusterNameKey] = clusterName
	result["SCOPE"] = string(reconcilerScope)
	result["POLICY_DIR"] = gitConfig.Dir
	result["GIT_REPO"] = gitConfig.Repo

	// Add Filesystem Polling Period.
	result[reconcilermanager.FilesystemPollingPeriod] = pollPeriod

	if gitConfig.Branch != "" {
		result["GIT_BRANCH"] = gitConfig.Branch
	} else {
		result["GIT_BRANCH"] = "master"
	}
	if gitConfig.Revision != "" {
		result["GIT_REV"] = gitConfig.Revision
	} else {
		result["GIT_REV"] = "HEAD"
	}
	return result
}

// sourceFormatData returns configmap for reconciler.
func sourceFormatData(format string) map[string]string {
	result := make(map[string]string)
	result[filesystem.SourceFormatKey] = format
	return result
}

func ownerReference(kind, name string, uid types.UID) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         v1alpha1.SchemeGroupVersion.String(),
		Kind:               kind,
		Name:               name,
		Controller:         pointer.BoolPtr(true),
		BlockOwnerDeletion: pointer.BoolPtr(true),
		UID:                uid,
	}
}

func envFromSources(configmapRef map[string]*bool) []corev1.EnvFromSource {
	var names []string
	for name := range configmapRef {
		names = append(names, name)
	}
	// We must sort the entries or else the Deployment's Pods will constantly get
	// reloaded due to random ordering of the spec.template.spec.envFrom field.
	sort.Strings(names)

	var envFromSource []corev1.EnvFromSource
	for _, name := range names {
		cfgMap := corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Optional: configmapRef[name],
			},
		}
		envFromSource = append(envFromSource, cfgMap)
	}
	return envFromSource
}
