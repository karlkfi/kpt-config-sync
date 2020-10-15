package status

import (
	"bytes"
	"testing"

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

func TestClusterState_PrintRows(t *testing.T) {
	testCases := []struct {
		name    string
		cluster *clusterState
		want    string
	}{
		{
			"cluster without repos",
			&clusterState{
				ref:    "gke_sample-project_europe-west1-b_cluster-1",
				status: "UNINSTALLED",
			},
			"--------------------\ngke_sample-project_europe-west1-b_cluster-1\nUNINSTALLED\n",
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
