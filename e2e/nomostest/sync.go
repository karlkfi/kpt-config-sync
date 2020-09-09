package nomostest

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
)

// RepoHasStatusSyncLatestToken ensures ACM has reported all objects were
// successfully synced to the repository.
func RepoHasStatusSyncLatestToken(sha1 string) Predicate {
	return func(o core.Object) error {
		repo, ok := o.(*v1.Repo)
		if !ok {
			return WrongTypeErr(o, &v1.Repo{})
		}

		// Ensure there aren't any pending changes to sync.
		if inProgress := len(repo.Status.Sync.InProgress); inProgress > 0 {
			return fmt.Errorf("status.sync.inProgress contains %d changes that haven't been synced", inProgress)
		}

		// Check the Sync.LatestToken as:
		// 1) Source.LatestToken is the most-recently-cloned hash of the git repository.
		//      It just means we've seen the update to the repository, but haven't
		//      updated the state of any objects on the cluster.
		// 2) Import.LatestToken is updated once we've successfully written the
		//      declared objects to ClusterConfigs/NamespaceConfigs, but haven't
		//      necessarily applied them to the cluster successfully.
		// 3) Sync.LatestToken is updated once we've updated the state of all
		//      objects on the cluster to match their declared states, so this is
		//      the one we want.
		if token := repo.Status.Sync.LatestToken; token != sha1 {
			return fmt.Errorf("status.sync.latestToken %q does not match git revision %q",
				token, sha1)
		}
		return nil
	}
}

// ClusterConfigHasSpecToken created a Predicate that ensures .spec.token on the
// passed ClusterConfig matches sha1.
//
// This means ACM is trying (or has tried) syncing cluster-scoped objects from
// the latest repo commit to the cluster.
func ClusterConfigHasSpecToken(sha1 string) Predicate {
	return func(o core.Object) error {
		cc, ok := o.(*v1.ClusterConfig)
		if !ok {
			return WrongTypeErr(o, &v1.ClusterConfig{})
		}

		if token := cc.Spec.Token; token != sha1 {
			return fmt.Errorf("spec.token %q does not match git revision %q",
				token, sha1)
		}
		return nil
	}
}

// ClusterConfigHasStatusToken created a Predicate that ensures .spec.token on the
// passed ClusterConfig matches sha1.
//
// This means ACM has successfully synced all cluster-scoped objects from the
// latest repo commit to the cluster.
func ClusterConfigHasStatusToken(sha1 string) Predicate {
	return func(o core.Object) error {
		cc, ok := o.(*v1.ClusterConfig)
		if !ok {
			return WrongTypeErr(o, &v1.ClusterConfig{})
		}

		if token := cc.Status.Token; token != sha1 {
			return fmt.Errorf("status.token %q does not match git revision %q",
				token, sha1)
		}
		return nil
	}
}

// RootSyncHasStatusSyncCommit creates a Predicate that ensures that the
// .status.sync.commit field on the passed RootSync matches sha1.
func RootSyncHasStatusSyncCommit(sha1 string) Predicate {
	return func(o core.Object) error {
		rs, ok := o.(*v1alpha1.RootSync)
		if !ok {
			return WrongTypeErr(o, &v1alpha1.RootSync{})
		}

		if errCount := len(rs.Status.Sync.Errors); errCount > 0 {
			return fmt.Errorf("status.sync.errors contains %d errors", errCount)
		}
		if commit := rs.Status.Sync.Commit; commit != sha1 {
			return fmt.Errorf("status.sync.commit %q does not match git revision %q", commit, sha1)
		}
		return nil
	}
}
