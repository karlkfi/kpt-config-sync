package status

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRepoState_PrintRows(t *testing.T) {
	testCases := []struct {
		name string
		repo *repoState
		want string
	}{
		{
			"optional git fields missing",
			&repoState{
				scope: "<root>",
				git: v1alpha1.Git{
					Repo: "https://github.com/tester/sample/",
				},
				status:    "SYNCED",
				commit:    "abc123",
				resources: exampleResources("abc123"),
			},
			"  <root>\thttps://github.com/tester/sample@master\t\n  SYNCED\tabc123\t\n  Managed resources:\n  \tNAMESPACE\tNAME\tSTATUS\tSOURCEHASH\n  \tbookstore\tdeployment.apps/test\tCurrent\tabc123\n  \tbookstore\tservice/test\tCurrent\tabc123\n  \tbookstore\tservice/test2\tConflict\tabc123\n",
		},
		{
			"optional git subdirectory specified",
			&repoState{
				scope: "<root>",
				git: v1alpha1.Git{
					Repo: "https://github.com/tester/sample/",
					Dir:  "quickstart//multirepo//root/",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  <root>\thttps://github.com/tester/sample/quickstart/multirepo/root@master\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git subdirectory is '/'",
			&repoState{
				scope: "<root>",
				git: v1alpha1.Git{
					Repo: "https://github.com/tester/sample/",
					Dir:  "/",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  <root>\thttps://github.com/tester/sample@master\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git subdirectory is '.'",
			&repoState{
				scope: "<root>",
				git: v1alpha1.Git{
					Repo: "https://github.com/tester/sample/",
					Dir:  ".",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  <root>\thttps://github.com/tester/sample@master\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git subdirectory starts with '/'",
			&repoState{
				scope: "<root>",
				git: v1alpha1.Git{
					Repo: "https://github.com/tester/sample/",
					Dir:  "/admin",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  <root>\thttps://github.com/tester/sample/admin@master\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git branch specified",
			&repoState{
				scope: "bookstore",
				git: v1alpha1.Git{
					Repo:   "https://github.com/tester/sample",
					Branch: "feature",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  bookstore\thttps://github.com/tester/sample@feature\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git revision specified",
			&repoState{
				scope: "bookstore",
				git: v1alpha1.Git{
					Repo:     "https://github.com/tester/sample",
					Revision: "v1",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  bookstore\thttps://github.com/tester/sample@v1\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional default git revision HEAD specified",
			&repoState{
				scope: "bookstore",
				git: v1alpha1.Git{
					Repo:     "https://github.com/tester/sample",
					Revision: "HEAD",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  bookstore\thttps://github.com/tester/sample@master\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional default git revision HEAD and branch specified",
			&repoState{
				scope: "bookstore",
				git: v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Revision: "HEAD",
					Branch:   "feature",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  bookstore\tgit@github.com:tester/sample@feature\t\n  SYNCED\tabc123\t\n",
		},
		{
			"all optional git fields specified",
			&repoState{
				scope: "bookstore",
				git: v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Dir:      "books",
					Branch:   "feature",
					Revision: "v1",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"  bookstore\tgit@github.com:tester/sample/books@v1\t\n  SYNCED\tabc123\t\n",
		},
		{
			"repo with errors",
			&repoState{
				scope: "bookstore",
				git: v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Dir:      "books",
					Revision: "v1",
				},
				status: "ERROR",
				commit: "abc123",
				errors: []string{"error1", "error2"},
			},
			"  bookstore\tgit@github.com:tester/sample/books@v1\t\n  ERROR\tabc123\t\n  Error:\terror1\t\n  Error:\terror2\t\n",
		},
		{
			"unsynced repo",
			&repoState{
				scope: "bookstore",
				git: v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Revision: "v1",
				},
				status: "PENDING",
			},
			"  bookstore\tgit@github.com:tester/sample@v1\t\n  PENDING\t\t\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			tc.repo.printRows(&buffer)
			got := buffer.String()
			if got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestRepoState_MonoRepoStatus(t *testing.T) {
	git := v1alpha1.Git{
		Repo:     "git@github.com:tester/sample",
		Revision: "v1",
		Dir:      "admin",
	}

	testCases := []struct {
		name   string
		git    v1alpha1.Git
		status v1.RepoStatus
		want   *repoState
	}{
		{
			"repo is pending first sync",
			git,
			v1.RepoStatus{
				Source: v1.RepoSourceStatus{},
				Import: v1.RepoImportStatus{},
				Sync:   v1.RepoSyncStatus{},
			},
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "PENDING",
				commit: "N/A",
			},
		},
		{
			"repo is synced",
			git,
			v1.RepoStatus{
				Source: v1.RepoSourceStatus{
					Token: "abc123",
				},
				Import: v1.RepoImportStatus{
					Token: "abc123",
				},
				Sync: v1.RepoSyncStatus{
					LatestToken: "abc123",
				},
			},
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "SYNCED",
				commit: "abc123",
			},
		},
		{
			"repo has errors",
			git,
			v1.RepoStatus{
				Source: v1.RepoSourceStatus{
					Token: "def456",
				},
				Import: v1.RepoImportStatus{
					Token: "def456",
					Errors: []v1.ConfigManagementError{
						{ErrorMessage: "KNV2010: I am unhappy"},
					},
				},
				Sync: v1.RepoSyncStatus{
					LatestToken: "abc123",
				},
			},
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2010: I am unhappy"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := monoRepoStatus(tc.git, tc.status)
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(*tc.want)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func withGitRepoSync(git v1alpha1.Git) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1alpha1.RepoSync)
		rs.Spec.Git = git
	}
}

func withCommitsRepoSync(rendering, source, sync string) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1alpha1.RepoSync)
		rs.Status.Rendering.Commit = rendering
		rs.Status.Source.Commit = source
		rs.Status.Sync.Commit = sync
	}
}

func withErrorsRepoSync(renderingErrs, sourceErrs, syncErrs []string) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1alpha1.RepoSync)
		for _, err := range renderingErrs {
			rs.Status.Rendering.Errors = append(rs.Status.Rendering.Errors, v1alpha1.ConfigSyncError{
				ErrorMessage: err,
			})
		}
		for _, err := range sourceErrs {
			rs.Status.Source.Errors = append(rs.Status.Source.Errors, v1alpha1.ConfigSyncError{
				ErrorMessage: err,
			})
		}
		for _, err := range syncErrs {
			rs.Status.Sync.Errors = append(rs.Status.Sync.Errors, v1alpha1.ConfigSyncError{
				ErrorMessage: err,
			})
		}
	}
}

func TestRepoState_NamespaceRepoStatus(t *testing.T) {
	git := v1alpha1.Git{
		Repo:     "git@github.com:tester/sample",
		Revision: "v1",
		Dir:      "admin",
	}

	testCases := []struct {
		name          string
		repoSync      *v1alpha1.RepoSync
		resourceGroup *unstructured.Unstructured
		want          *repoState
	}{
		{
			"repo is pending first sync",
			fake.RepoSyncObject(core.Namespace("bookstore"), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync")),
			&repoState{
				scope:  "bookstore",
				git:    git,
				status: "PENDING",
				commit: "N/A",
			},
		},
		{
			"repo is synced before automated-hydration is supported",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("", "abc123", "abc123"), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync"), withResources()),
			&repoState{
				scope:     "bookstore",
				git:       git,
				status:    "SYNCED",
				commit:    "abc123",
				resources: exampleResources(""),
			},
		},
		{
			"repo is synced",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("abc123", "abc123", "abc123"), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync"), withResources()),
			&repoState{
				scope:     "bookstore",
				git:       git,
				status:    "SYNCED",
				commit:    "abc123",
				resources: exampleResources(""),
			},
		},
		{
			"repo is rendering new commit",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("def456", "abc123", "abc123"), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync"), withResourcesAndCommit("abc123")),
			&repoState{
				scope:     "bookstore",
				git:       git,
				status:    "PENDING",
				commit:    "abc123",
				resources: exampleResources("abc123"),
			},
		},
		{
			"repo has import errors",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("abc123", "def456", "abc123"), withErrorsRepoSync([]string{}, []string{"KNV2004: import error"}, []string{}), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync")),
			&repoState{
				scope:  "bookstore",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2004: import error"},
			},
		},
		{
			"repo has rendering errors",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("def456", "abc123", "abc123"), withErrorsRepoSync([]string{"KNV2015: rendering error"}, []string{}, []string{}), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync")),
			&repoState{
				scope:  "bookstore",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2015: rendering error"},
			},
		},
		{
			"repo has parsing errors",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("def456", "def456", "abc123"), withErrorsRepoSync([]string{}, []string{"KNV2004: parsing error"}, []string{}), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync")),
			&repoState{
				scope:  "bookstore",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2004: parsing error"},
			},
		},
		{
			"repo has sync errors",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("abc123", "abc123", "abc123"), withErrorsRepoSync([]string{}, []string{}, []string{"KNV2009: apply error"}), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync")),
			&repoState{
				scope:  "bookstore",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2009: apply error"},
			},
		},
		{
			"repo is in a weird state: rendering.commit != source.commit, but rendering.commit == sync.commit without any errors",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("abc123", "abc456", "abc123"), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync")),
			&repoState{
				scope:  "bookstore",
				git:    git,
				status: "PENDING",
				commit: "abc123",
			},
		},
		{
			"repo is in a weird state: rendering.commit != source.commit, and rendering.commit != sync.commit without any errors",
			fake.RepoSyncObject(core.Namespace("bookstore"), withCommitsRepoSync("abc123", "abc456", "abc000"), withGitRepoSync(git)),
			fake.ResourceGroupObject(core.Namespace("bookstore"), core.Name("repo-sync")),
			&repoState{
				scope:  "bookstore",
				git:    git,
				status: "PENDING",
				commit: "abc000",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := namespaceRepoStatus(tc.repoSync, tc.resourceGroup)
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(*tc.want)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func withGitRootSync(git v1alpha1.Git) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1alpha1.RootSync)
		rs.Spec.Git = git
	}
}

func withCommitsRootSync(rendering, source, sync string) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1alpha1.RootSync)
		rs.Status.Rendering.Commit = rendering
		rs.Status.Source.Commit = source
		rs.Status.Sync.Commit = sync
	}
}

func withErrorsRootSync(renderingErrs, sourceErrs, syncErrs []string) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1alpha1.RootSync)
		for _, err := range renderingErrs {
			rs.Status.Rendering.Errors = append(rs.Status.Rendering.Errors, v1alpha1.ConfigSyncError{
				ErrorMessage: err,
			})
		}
		for _, err := range sourceErrs {
			rs.Status.Source.Errors = append(rs.Status.Source.Errors, v1alpha1.ConfigSyncError{
				ErrorMessage: err,
			})
		}
		for _, err := range syncErrs {
			rs.Status.Sync.Errors = append(rs.Status.Sync.Errors, v1alpha1.ConfigSyncError{
				ErrorMessage: err,
			})
		}
	}
}

func TestRepoState_RootRepoStatus(t *testing.T) {
	git := v1alpha1.Git{
		Repo:     "git@github.com:tester/sample",
		Revision: "v1",
		Dir:      "admin",
	}

	testCases := []struct {
		name     string
		rootSync *v1alpha1.RootSync
		want     *repoState
	}{
		{
			"repo is pending first sync",
			fake.RootSyncObject(withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "PENDING",
				commit: "N/A",
			},
		},
		{
			"repo is synced before automated-hydration is supported",
			fake.RootSyncObject(withCommitsRootSync("", "abc123", "abc123"), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "SYNCED",
				commit: "abc123",
			},
		},
		{
			"repo is synced",
			fake.RootSyncObject(withCommitsRootSync("abc123", "abc123", "abc123"), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "SYNCED",
				commit: "abc123",
			},
		},
		{
			"repo is rendering a new commit",
			fake.RootSyncObject(withCommitsRootSync("def456", "abc123", "abc123"), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "PENDING",
				commit: "abc123",
			},
		},
		{
			"repo has import errors",
			fake.RootSyncObject(withCommitsRootSync("abc123", "def456", "abc123"), withErrorsRootSync([]string{}, []string{"KNV2004: import error"}, []string{}), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2004: import error"},
			},
		},
		{
			"repo has rendering errors",
			fake.RootSyncObject(withCommitsRootSync("def456", "abc123", "abc123"), withErrorsRootSync([]string{"KNV2015: rendering error"}, []string{}, []string{}), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2015: rendering error"},
			},
		},
		{
			"repo has parsing errors",
			fake.RootSyncObject(withCommitsRootSync("def456", "def456", "abc123"), withErrorsRootSync([]string{}, []string{"KNV2004: parse error"}, []string{}), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2004: parse error"},
			},
		},
		{
			"repo has sync errors",
			fake.RootSyncObject(withCommitsRootSync("abc123", "abc123", "abc123"), withErrorsRootSync([]string{}, []string{}, []string{"KNV2009: apply error"}), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "ERROR",
				commit: "abc123",
				errors: []string{"KNV2009: apply error"},
			},
		},
		{
			"repo is in a weird state: rendering.commit != source.commit, but rendering.commit == sync.commit without any errors",
			fake.RootSyncObject(withCommitsRootSync("abc123", "abc456", "abc123"), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "PENDING",
				commit: "abc123",
			},
		},
		{
			"repo is in a weird state: rendering.commit != source.commit, and rendering.commit != sync.commit without any errors",
			fake.RootSyncObject(withCommitsRootSync("abc123", "abc456", "abc000"), withGitRootSync(git)),
			&repoState{
				scope:  "<root>",
				git:    git,
				status: "PENDING",
				commit: "abc000",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := rootRepoStatus(tc.rootSync, nil)
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(*tc.want)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestClusterState_PrintRows(t *testing.T) {
	testCases := []struct {
		name    string
		cluster *clusterState
		want    string
	}{
		{
			"cluster without config sync",
			&clusterState{
				ref:    "gke_sample-project_europe-west1-b_cluster-1",
				status: "UNINSTALLED",
			},
			`
gke_sample-project_europe-west1-b_cluster-1
  --------------------
  UNINSTALLED	
`,
		},
		{
			"cluster without repos",
			&clusterState{
				ref:    "gke_sample-project_europe-west1-b_cluster-1",
				status: "UNCONFIGURED",
				error:  "Missing git-creds secret",
			},
			`
gke_sample-project_europe-west1-b_cluster-1
  --------------------
  UNCONFIGURED	Missing git-creds secret
`,
		},
		{
			"cluster with repos",
			&clusterState{
				ref: "gke_sample-project_europe-west1-b_cluster-2",
				repos: []*repoState{
					{
						scope: "<root>",
						git: v1alpha1.Git{
							Repo: "git@github.com:tester/sample",
						},
						status: "SYNCED",
						commit: "abc123",
					},
					{
						scope: "bookstore",
						git: v1alpha1.Git{
							Repo:   "git@github.com:tester/sample",
							Branch: "feature",
						},
						status: "SYNCED",
						commit: "abc123",
					},
				},
			},
			`
gke_sample-project_europe-west1-b_cluster-2
  --------------------
  <root>	git@github.com:tester/sample@master	
  SYNCED	abc123	
  --------------------
  bookstore	git@github.com:tester/sample@feature	
  SYNCED	abc123	
`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			tc.cluster.printRows(&buffer)
			got := buffer.String()
			if got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func withResources() core.MetaMutator {
	status := map[string]interface{}{
		"resourceStatuses": []interface{}{
			map[string]interface{}{
				"group":     "apps",
				"kind":      "Deployment",
				"namespace": "bookstore",
				"name":      "test",
				"status":    "Current",
			},
			map[string]interface{}{
				"kind":      "Service",
				"namespace": "bookstore",
				"name":      "test",
				"status":    "Current",
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Stalled",
						"status": "False",
					},
				},
			},
			map[string]interface{}{
				"kind":      "Service",
				"namespace": "bookstore",
				"name":      "test2",
				"status":    "Current",
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "OwnershipOverlap",
						"status": "True",
					},
				},
			},
		},
	}
	return func(o client.Object) {
		u := o.(*unstructured.Unstructured)
		unstructured.SetNestedField(u.Object, status, "status") //nolint
	}
}

func withResourcesAndCommit(commit string) core.MetaMutator {
	status := map[string]interface{}{
		"resourceStatuses": []interface{}{
			map[string]interface{}{
				"group":      "apps",
				"kind":       "Deployment",
				"namespace":  "bookstore",
				"name":       "test",
				"status":     "Current",
				"sourceHash": commit,
			},
			map[string]interface{}{
				"kind":       "Service",
				"namespace":  "bookstore",
				"name":       "test",
				"status":     "Current",
				"sourceHash": commit,
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Stalled",
						"status": "False",
					},
				},
			},
			map[string]interface{}{
				"kind":       "Service",
				"namespace":  "bookstore",
				"name":       "test2",
				"status":     "Current",
				"sourceHash": commit,
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "OwnershipOverlap",
						"status": "True",
					},
				},
			},
		},
	}
	return func(o client.Object) {
		u := o.(*unstructured.Unstructured)
		unstructured.SetNestedField(u.Object, status, "status") //nolint
	}
}

func exampleResources(commit string) []resourceState {
	return []resourceState{
		{
			Group:      "apps",
			Kind:       "Deployment",
			Namespace:  "bookstore",
			Name:       "test",
			Status:     "Current",
			SourceHash: commit,
		},
		{
			Group:      "",
			Kind:       "Service",
			Namespace:  "bookstore",
			Name:       "test",
			Status:     "Current",
			SourceHash: commit,
			Conditions: []Condition{{Type: "Stalled", Status: "False"}},
		},
		{
			Group:      "",
			Kind:       "Service",
			Namespace:  "bookstore",
			Name:       "test2",
			Status:     "Conflict",
			SourceHash: commit,
			Conditions: []Condition{{Type: "OwnershipOverlap", Status: "True"}},
		},
	}
}
