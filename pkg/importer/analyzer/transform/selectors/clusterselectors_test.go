package selectors

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors/seltest"
	"github.com/google/nomos/pkg/object"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type clusterselectorsTestCase struct {
	name               string
	clusters           []clusterregistry.Cluster
	selectors          []v1.ClusterSelector
	expectedMapping    ClusterSelectors
	expectedMatches    []object.Annotated
	expectedMismatches []object.Annotated
	expectedForEach    map[string]v1.ClusterSelector
}

func (tc *clusterselectorsTestCase) run(t *testing.T) {
	s, err := NewClusterSelectors(tc.clusters, tc.selectors, "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(tc.expectedMapping, *s) {
		t.Errorf("GetClusterSelectors(_)=%+v,\nwant:\n%+v\ndiff:\n%v", *s, tc.expectedMapping, cmp.Diff(tc.expectedMapping, *s))
	}
	for _, o := range tc.expectedMatches {
		if !tc.expectedMapping.Matches(o) {
			t.Errorf("Matches(%+v)=false, expected true: for: %+v", o, tc.expectedMapping)
		}
	}
	for _, o := range tc.expectedMismatches {
		if tc.expectedMapping.Matches(o) {
			t.Errorf("Matches(%+v)=true, expected false: for: %+v", o, tc.expectedMapping)
		}
	}
	if tc.expectedForEach != nil {
		m := map[string]v1.ClusterSelector{}
		s.ForEachSelector(func(name string, selector v1.ClusterSelector) {
			m[name] = selector
		})
		if !cmp.Equal(m, s.selectors) {
			t.Errorf("Selector map mismatch: diff:\n%v", cmp.Diff(m, s.selectors))
		}
	}
}

func TestVisitor(t *testing.T) {
	tests := []clusterselectorsTestCase{
		{
			name:     "Only cluster list",
			clusters: []clusterregistry.Cluster{},
			expectedMapping: ClusterSelectors{
				clusterName: "cluster-1",
				selectors:   map[string]v1.ClusterSelector{},
			},
			expectedMatches: []object.Annotated{
				// An un-annotated thing matches always.
				seltest.Annotated(map[string]string{}),
			},
		},
		{
			name:      "Only selector list",
			selectors: []v1.ClusterSelector{},
			expectedMapping: ClusterSelectors{
				clusterName: "cluster-1",
				selectors:   map[string]v1.ClusterSelector{},
			},
			expectedMatches: []object.Annotated{
				seltest.Annotated(map[string]string{}),
			},
		},
		{
			name: "Basic",
			clusters: []clusterregistry.Cluster{
				seltest.Cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
			},
			selectors: []v1.ClusterSelector{
				seltest.Selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "prod",
						},
					}),
			},
			expectedMapping: ClusterSelectors{
				clusterName: "cluster-1",
				selectors: map[string]v1.ClusterSelector{
					"sel-1": seltest.Selector("sel-1",
						metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "prod",
							},
						}),
				},
				cluster: seltest.Cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
			},
			expectedMatches: []object.Annotated{
				seltest.Annotated(map[string]string{}),
				seltest.Annotated(map[string]string{
					v1.ClusterSelectorAnnotationKey: "sel-1",
				}),
			},
			expectedMismatches: []object.Annotated{
				seltest.Annotated(map[string]string{
					v1.ClusterSelectorAnnotationKey: "sel-2",
				}),
			},
		},
		{
			name: "Mismatching labels",
			clusters: []clusterregistry.Cluster{
				seltest.Cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
			},
			selectors: []v1.ClusterSelector{
				seltest.Selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "test",
						},
					}),
			},
			expectedMapping: ClusterSelectors{
				clusterName: "cluster-1",
				cluster: seltest.Cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
				selectors: map[string]v1.ClusterSelector{},
			},
			expectedMatches: []object.Annotated{
				seltest.Annotated(map[string]string{}),
			},
			expectedMismatches: []object.Annotated{
				seltest.Annotated(map[string]string{
					v1.ClusterSelectorAnnotationKey: "sel-1",
				}),
				seltest.Annotated(map[string]string{
					v1.ClusterSelectorAnnotationKey: "sel-2",
				}),
				seltest.Annotated(map[string]string{
					v1.ClusterSelectorAnnotationKey: "unknown-selector",
				}),
			},
		},
		{
			name: "Unlabeled cluster matches any selector",
			clusters: []clusterregistry.Cluster{
				seltest.Cluster("cluster-1", map[string]string{}),
			},
			selectors: []v1.ClusterSelector{
				seltest.Selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "test",
						},
					}),
			},
			expectedMapping: ClusterSelectors{
				clusterName: "cluster-1",
				cluster:     seltest.Cluster("cluster-1", map[string]string{}),
				selectors:   map[string]v1.ClusterSelector{},
			},
			expectedForEach: map[string]v1.ClusterSelector{
				"sel-1": seltest.Selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "test",
						},
					}),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}
