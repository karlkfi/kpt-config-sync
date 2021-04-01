package controllers

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
)

// GitConfig constants.
const (
	// DefaultSyncRev is the default git revision.
	DefaultSyncRev = "HEAD"
	// SyncDepthNoRev is the default git depth if syncing with default sync revision (`HEAD`).
	SyncDepthNoRev = "1"
	// SyncDepthRev is the default git depth if syncing with a specific sync revision (tag or hash).
	SyncDepthRev = "500"
)

var gceNodeAskpassURL = fmt.Sprintf("http://localhost:%v/git_askpass", gceNodeAskpassPort)

type options struct {
	// ref is the git revision being synced.
	ref string
	// branch is the git branch being synced.
	branch string
	// repo is the git repo being synced.
	repo string
	// secretType used to connect to the repo.
	secretType string
	// proxy used to connect to the repo.
	proxy string
	// period is the time in seconds between consecutive syncs.
	period float64
}

func gitSyncData(opts options) map[string]string {
	result := make(map[string]string)
	result["GIT_SYNC_REPO"] = opts.repo
	result["GIT_KNOWN_HOSTS"] = "false" // disable known_hosts checking because it provides no benefit for our use case.
	// If syncRev is set, git-sync checks out the source repo at master and then resets to
	// the specified rev. This means that the rev has to be in the pulled history and thus
	// will fail if rev is earlier than the configured depth.
	// However, if history is too large git-sync will OOM when it tries to pull all of it.
	// Try to set a happy medium here -- if syncRev is set, pull 500 commits from master;
	// if it isn't, just the latest commit will do and will save memory.
	// See b/175088702 and b/158988143
	if opts.ref == "" || opts.ref == DefaultSyncRev {
		result["GIT_SYNC_DEPTH"] = SyncDepthNoRev
	} else {
		result["GIT_SYNC_DEPTH"] = SyncDepthRev
	}
	result["GIT_SYNC_WAIT"] = fmt.Sprintf("%f", opts.period)
	// When branch and ref not set in RootSync/RepoSync then dont set GIT_SYNC_BRANCH
	// and GIT_SYNC_REV, git-sync will use the default values for them.
	if opts.branch != "" {
		result["GIT_SYNC_BRANCH"] = opts.branch
	}
	if opts.ref != "" {
		result["GIT_SYNC_REV"] = opts.ref
	}
	switch opts.secretType {
	case v1alpha1.GitSecretGCENode, v1alpha1.GitSecretGCPServiceAccount:
		result["GIT_ASKPASS_URL"] = gceNodeAskpassURL
	case v1alpha1.GitSecretSSH:
		result["GIT_SYNC_SSH"] = "true"
	case v1alpha1.GitSecretCookieFile:
		result["GIT_COOKIE_FILE"] = "true"

		fallthrough
	case v1alpha1.GitSecretConfigKeyToken, "", v1alpha1.GitSecretNone:
		result["HTTPS_PROXY"] = opts.proxy
	default:
		// TODO b/168553377 Return error while setting up gitSyncData.
		glog.Errorf("Unrecognized secret type %s", opts.secretType)
	}
	return result
}
