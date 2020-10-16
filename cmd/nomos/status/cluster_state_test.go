package status

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
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
				git: &v1alpha1.Git{
					Repo: "git@github.com:tester/sample",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"<root>\tgit@github.com:tester/sample@master\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git subdirectory specified",
			&repoState{
				scope: "<root>",
				git: &v1alpha1.Git{
					Repo: "git@github.com:tester/sample",
					Dir:  "admin",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"<root>\tgit@github.com:tester/sample/admin@master\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git branch specified",
			&repoState{
				scope: "bookstore",
				git: &v1alpha1.Git{
					Repo:   "git@github.com:tester/sample",
					Branch: "feature",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"bookstore\tgit@github.com:tester/sample@feature\t\n  SYNCED\tabc123\t\n",
		},
		{
			"optional git revision specified",
			&repoState{
				scope: "bookstore",
				git: &v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Revision: "v1",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"bookstore\tgit@github.com:tester/sample@v1\t\n  SYNCED\tabc123\t\n",
		},
		{
			"all optional git fields specified",
			&repoState{
				scope: "bookstore",
				git: &v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Dir:      "books",
					Branch:   "feature",
					Revision: "v1",
				},
				status: "SYNCED",
				commit: "abc123",
			},
			"bookstore\tgit@github.com:tester/sample/books@v1\t\n  SYNCED\tabc123\t\n",
		},
		{
			"repo with errors",
			&repoState{
				scope: "bookstore",
				git: &v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Dir:      "books",
					Revision: "v1",
				},
				status: "ERROR",
				commit: "abc123",
				errors: []string{"error1", "error2"},
			},
			"bookstore\tgit@github.com:tester/sample/books@v1\t\n  ERROR\tabc123\t\n  Error:\terror1\t\n  Error:\terror2\t\n",
		},
		{
			"unsynced repo",
			&repoState{
				scope: "bookstore",
				git: &v1alpha1.Git{
					Repo:     "git@github.com:tester/sample",
					Revision: "v1",
				},
				status: "PENDING",
			},
			"bookstore\tgit@github.com:tester/sample@v1\t\n  PENDING\t\t\n",
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
	git := &v1alpha1.Git{
		Repo:     "git@github.com:tester/sample",
		Revision: "v1",
		Dir:      "admin",
	}

	testCases := []struct {
		name   string
		git    *v1alpha1.Git
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
			"--------------------\ngke_sample-project_europe-west1-b_cluster-1\nUNINSTALLED\t\n",
		},
		{
			"cluster without repos",
			&clusterState{
				ref:    "gke_sample-project_europe-west1-b_cluster-1",
				status: "UNCONFIGURED",
				error:  "Missing git-creds secret",
			},
			"--------------------\ngke_sample-project_europe-west1-b_cluster-1\nUNCONFIGURED\tMissing git-creds secret\n",
		},
		{
			"cluster with repos",
			&clusterState{
				ref: "gke_sample-project_europe-west1-b_cluster-2",
				repos: []*repoState{
					{
						scope: "<root>",
						git: &v1alpha1.Git{
							Repo: "git@github.com:tester/sample",
						},
						status: "SYNCED",
						commit: "abc123",
					},
					{
						scope: "bookstore",
						git: &v1alpha1.Git{
							Repo:   "git@github.com:tester/sample",
							Branch: "feature",
						},
						status: "SYNCED",
						commit: "abc123",
					},
				},
			},
			"--------------------\ngke_sample-project_europe-west1-b_cluster-2\n<root>\tgit@github.com:tester/sample@master\t\n  SYNCED\tabc123\t\nbookstore\tgit@github.com:tester/sample@feature\t\n  SYNCED\tabc123\t\n",
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
