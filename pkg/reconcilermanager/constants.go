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

package reconcilermanager

const (
	// ManagerName is the name of the controller which creates reconcilers.
	ManagerName = "reconciler-manager"
)

const (
	// SourceFormat is the key used for storing whether a repository is
	// unstructured or in hierarchy mode. Used in many objects related to this
	// behavior.
	SourceFormat = "source-format"

	// ClusterNameKey is the OS env variable and ConfigMap key for the name
	// of the cluster.
	ClusterNameKey = "CLUSTER_NAME"

	// ScopeKey is the OS env variable and ConfigMap key for the scope of the
	// reconciler and hydration controller.
	ScopeKey = "SCOPE"

	// SyncNameKey is the OS env variable and ConfigMap key for the name of
	// the RootSync or RepoSync object.
	SyncNameKey = "SYNC_NAME"

	// ReconcilerNameKey is the OS env variable and ConfigMap key for the name of
	// the Reconciler Deployment.
	ReconcilerNameKey = "RECONCILER_NAME"

	// SyncDirKey is the OS env variable and ConfigMap key for the sync directory
	// read by the hydration controller.
	SyncDirKey = "SYNC_DIR"

	// PolicyDirKey is the OS env variable and ConfigMap key for the sync directory
	// read by the reconciler.
	PolicyDirKey = "POLICY_DIR"

	// GitSync is the name of the git-sync container in reconciler pods.
	GitSync = "git-sync"

	// HydrationController is the name of the hydration-controller container in reconciler pods.
	HydrationController = "hydration-controller"

	// Reconciler is a common building block for many resource names associated
	// with reconciling resources.
	Reconciler = "reconciler"

	// StatusMode is to control if the kpt applier needs to inject the actuation data
	// into the ResourceGroup object.
	StatusMode = "STATUS_MODE"
)

const (
	// GitRepoKey is the OS env variable and ConfigMap key for the git repo URL.
	GitRepoKey = "GIT_REPO"

	// GitBranchKey is the OS env variable and ConfigMap key for the git branch name.
	GitBranchKey = "GIT_BRANCH"

	// GitRevKey is the OS env variable and ConfigMap key for the git revision.
	GitRevKey = "GIT_REV"
)

const (
	// ReconcilerPollingPeriod defines how often the reconciler should poll the
	// filesystem for updates to the source or rendered configs.
	ReconcilerPollingPeriod = "RECONCILER_POLLING_PERIOD"

	// HydrationPollingPeriod defines how often the hydration controller should
	// poll the filesystem for rendering the DRY configs.
	HydrationPollingPeriod = "HYDRATION_POLLING_PERIOD"
)
