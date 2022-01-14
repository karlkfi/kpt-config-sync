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
