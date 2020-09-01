package controllers

import (
	"sort"

	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// gitSyncData returns configmap for git-sync container.
func gitSyncData(ref, branch, repo string) map[string]string {
	result := make(map[string]string)
	result["GIT_KNOWN_HOSTS"] = "false"
	result["GIT_SYNC_REPO"] = repo
	result["GIT_SYNC_WAIT"] = "15"
	// When branch and ref not set in RootSync/RepoSync then dont set GIT_SYNC_BRANCH
	// and GIT_SYNC_REV, git-sync will use the default values for them.
	if branch != "" {
		result["GIT_SYNC_BRANCH"] = branch
	}
	if ref != "" {
		result["GIT_SYNC_REV"] = ref
	}
	return result
}

// reconcilerData returns configmap data for namespace reconciler.
func reconcilerData(reconcilerScope declared.Scope, policyDir, gitRepo, gitBranch, gitRev string) map[string]string {
	result := make(map[string]string)
	result["SCOPE"] = string(reconcilerScope)
	result["POLICY_DIR"] = policyDir
	result["GIT_REPO"] = gitRepo
	if gitBranch != "" {
		result["GIT_BRANCH"] = gitBranch
	} else {
		result["GIT_BRANCH"] = "master"
	}
	if gitRev != "" {
		result["GIT_REV"] = gitRev
	} else {
		result["GIT_REV"] = "HEAD"
	}
	return result
}

// rootReconcilerData returns configmap data for root reconciler.
func rootReconcilerData(reconcilerScope declared.Scope, policyDir, clusterName, gitRepo, gitBranch, gitRev string) map[string]string {
	result := reconcilerData(reconcilerScope, policyDir, gitRepo, gitBranch, gitRev)
	result["CLUSTER_NAME"] = clusterName
	return result
}

// sourceFormatData returns configmap for reconciler.
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
