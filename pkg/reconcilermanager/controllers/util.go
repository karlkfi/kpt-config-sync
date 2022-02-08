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
	"os"
	"sort"
	"time"

	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// hydrationData returns configmap data for the hydration controller.
func hydrationData(gitConfig *v1beta1.Git, scope declared.Scope, reconcilerName, pollPeriod string) map[string]string {
	result := make(map[string]string)
	result[reconcilermanager.ScopeKey] = string(scope)
	result[reconcilermanager.ReconcilerNameKey] = reconcilerName
	result[reconcilermanager.SyncDirKey] = gitConfig.Dir
	// Add Hydration Polling Period.
	result[reconcilermanager.HydrationPollingPeriod] = pollPeriod
	return result
}

// reconcilerData returns configmap data for namespace reconciler.
func reconcilerData(clusterName, syncName, reconcilerName string, reconcilerScope declared.Scope, gitConfig *v1beta1.Git, pollPeriod string) map[string]string {
	result := make(map[string]string)
	result[reconcilermanager.ClusterNameKey] = clusterName
	result[reconcilermanager.ScopeKey] = string(reconcilerScope)
	result[reconcilermanager.SyncNameKey] = syncName
	result[reconcilermanager.ReconcilerNameKey] = reconcilerName
	result[reconcilermanager.PolicyDirKey] = gitConfig.Dir
	result[reconcilermanager.GitRepoKey] = gitConfig.Repo

	// Add Filesystem Polling Period.
	result[reconcilermanager.ReconcilerPollingPeriod] = pollPeriod

	if gitConfig.Branch != "" {
		result[reconcilermanager.GitBranchKey] = gitConfig.Branch
	} else {
		result[reconcilermanager.GitBranchKey] = "master"
	}
	if gitConfig.Revision != "" {
		result[reconcilermanager.GitRevKey] = gitConfig.Revision
	} else {
		result[reconcilermanager.GitRevKey] = "HEAD"
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
		APIVersion:         v1beta1.SchemeGroupVersion.String(),
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

// PollingPeriod parses the polling duration from the environment variable.
// If the variable is not present, it returns the default value.
func PollingPeriod(envName string, defaultValue time.Duration) time.Duration {
	val, present := os.LookupEnv(envName)
	if present {
		pollingFreq, err := time.ParseDuration(val)
		if err != nil {
			panic(errors.Wrapf(err, "failed to parse environment variable %q,"+
				"got value: %v, want err: nil", envName, pollingFreq))
		}
		return pollingFreq
	}
	return defaultValue
}
