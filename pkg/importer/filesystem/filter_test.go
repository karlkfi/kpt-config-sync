package filesystem

import (
	"path"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

func toCmpath(files []string) []cmpath.Path {
	var result []cmpath.Path
	for _, f := range files {
		result = append(result, cmpath.FromSlash(f))
	}
	return result
}

func fromCmpath(paths []cmpath.Path) []string {
	var result []string
	for _, p := range paths {
		result = append(result, p.SlashPath())
	}
	return result
}

func TestFilterHierarchyFiles(t *testing.T) {
	testCases := []struct {
		name  string
		root  string
		files []string
		want  []string
	}{
		{
			name: "empty works",
		},
		{
			name:  "keep system/",
			root:  "/",
			files: []string{path.Join("/", repo.SystemDir, "repo.yaml")},
			want:  []string{path.Join("/", repo.SystemDir, "repo.yaml")},
		},
		{
			name:  "keep cluster/",
			root:  "/",
			files: []string{path.Join("/", repo.ClusterDir, "cr.yaml")},
			want:  []string{path.Join("/", repo.ClusterDir, "cr.yaml")},
		},
		{
			name:  "keep clusterregistry/",
			root:  "/",
			files: []string{path.Join("/", repo.ClusterRegistryDir, "cluster.yaml")},
			want:  []string{path.Join("/", repo.ClusterRegistryDir, "cluster.yaml")},
		},
		{
			name:  "keep namespaces/",
			root:  "/",
			files: []string{path.Join("/", repo.NamespacesDir, "ns.yaml")},
			want:  []string{path.Join("/", repo.NamespacesDir, "ns.yaml")},
		},
		{
			name:  "ignore top-level",
			root:  "/",
			files: []string{"namespaces.yaml"},
		},
		{
			name:  "ignore other subdirectory",
			root:  "/",
			files: []string{path.Join("other", "repo.yaml")},
		},
		{
			name:  "ignore other subdirectory",
			root:  "/",
			files: []string{path.Join("other", "repo.yaml")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := fromCmpath(FilterHierarchyFiles(cmpath.FromSlash(tc.root), toCmpath(tc.files)))

			sort.Strings(tc.want)
			sort.Strings(got)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}
