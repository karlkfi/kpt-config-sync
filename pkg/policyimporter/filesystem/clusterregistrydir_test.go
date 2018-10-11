package filesystem

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	policyhierarchy "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	fstesting "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

type clusterRegistryDirTestCase struct {
	name string

	resources []*resource.Info

	wantErr       bool
	wantClusters  []clusterregistry.Cluster
	wantSelectors []policyhierarchy.ClusterSelector
}

func (tc *clusterRegistryDirTestCase) Run(t *testing.T) {
	crc, css, err := processClusterRegistryDir(clusterregistryDir, tc.resources)
	if err != nil {
		if tc.wantErr {
			return // Expected
		}
		t.Errorf("unexpected error: %v", err)
	}
	if !cmp.Equal(tc.wantClusters, crc) {
		t.Errorf("want clusters:\n%v,\ngot clusters:\n%v,\ndiff:\n%v", tc.wantClusters, crc, cmp.Diff(tc.wantClusters, crc))
	}
	if !cmp.Equal(tc.wantSelectors, css) {
		t.Errorf("want selectors:\n%v,\ngot selectors:\n%v,\ndiff:\n%v", tc.wantSelectors, css, cmp.Diff(tc.wantSelectors, css))
	}
}

func TestClusterRegistryDir(t *testing.T) {
	f := fstesting.NewTestFactory()
	defer f.Cleanup()
	tests := []clusterRegistryDirTestCase{
		{
			name: "Empty",
		},
		{
			name:    "Disallowed object type present",
			wantErr: true,
			resources: []*resource.Info{
				f.ResourceInfo(&policyhierarchy.Sync{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchy.SchemeGroupVersion.String(),
						Kind:       "Sync",
					},
				}),
			},
		},
		{
			name: "Converted cluster selector",
			resources: []*resource.Info{
				f.ResourceInfo(&policyhierarchy.ClusterSelector{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchy.SchemeGroupVersion.String(),
						Kind:       "ClusterSelector",
					},
				}),
			},
			wantSelectors: []policyhierarchy.ClusterSelector{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchy.SchemeGroupVersion.String(),
						Kind:       "ClusterSelector",
					},
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"nomos.dev/declaration-path": "clusterregistry",
						},
					},
				},
			},
		},
		{
			name: "Converted cluster",
			resources: []*resource.Info{
				f.ResourceInfo(&clusterregistry.Cluster{
					TypeMeta: metav1.TypeMeta{
						APIVersion: clusterregistry.SchemeGroupVersion.String(),
						Kind:       "Cluster",
					},
				}),
			},
			wantClusters: []clusterregistry.Cluster{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: clusterregistry.SchemeGroupVersion.String(),
						Kind:       "Cluster",
					},
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"nomos.dev/declaration-path": "clusterregistry",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.Run)
	}
}
