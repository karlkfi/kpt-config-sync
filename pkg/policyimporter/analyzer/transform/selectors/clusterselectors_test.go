/*
Copyright 2018 The Nomos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package selectors

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	policyhierarchy "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors/seltest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type clusterselectorsTestCase struct {
	name               string
	clusters           []clusterregistry.Cluster
	selectors          []policyhierarchy.ClusterSelector
	expectedMapping    ClusterSelectors
	expectedMatches    []ast.Annotated
	expectedMismatches []ast.Annotated
	expectedForEach    map[string]policyhierarchy.ClusterSelector
}

func Opts() cmp.Options {
	return cmp.Options{
		cmp.AllowUnexported(ClusterSelectors{}),
	}
}

func (tc *clusterselectorsTestCase) run(t *testing.T) {
	s, err := NewClusterSelectors(tc.clusters, tc.selectors, "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(tc.expectedMapping, *s, Opts()...) {
		t.Errorf("GetClusterSelectors(_)=%+v,\nwant:\n%+v\ndiff:\n%v", *s, tc.expectedMapping, cmp.Diff(tc.expectedMapping, *s, Opts()))
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
		m := map[string]policyhierarchy.ClusterSelector{}
		s.ForEachSelector(func(name string, selector policyhierarchy.ClusterSelector) {
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
				selectors:   map[string]policyhierarchy.ClusterSelector{},
			},
			expectedMatches: []ast.Annotated{
				// An un-annotated thing matches always.
				seltest.Annotated(map[string]string{}),
			},
		},
		{
			name:      "Only selector list",
			selectors: []policyhierarchy.ClusterSelector{},
			expectedMapping: ClusterSelectors{
				clusterName: "cluster-1",
				selectors:   map[string]policyhierarchy.ClusterSelector{},
			},
			expectedMatches: []ast.Annotated{
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
			selectors: []policyhierarchy.ClusterSelector{
				seltest.Selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "prod",
						},
					}),
			},
			expectedMapping: ClusterSelectors{
				clusterName: "cluster-1",
				selectors: map[string]policyhierarchy.ClusterSelector{
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
			expectedMatches: []ast.Annotated{
				seltest.Annotated(map[string]string{}),
				seltest.Annotated(map[string]string{
					policyhierarchy.ClusterSelectorAnnotationKey: "sel-1",
				}),
			},
			expectedMismatches: []ast.Annotated{
				seltest.Annotated(map[string]string{
					policyhierarchy.ClusterSelectorAnnotationKey: "sel-2",
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
			selectors: []policyhierarchy.ClusterSelector{
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
				selectors: map[string]policyhierarchy.ClusterSelector{},
			},
			expectedMatches: []ast.Annotated{
				seltest.Annotated(map[string]string{}),
			},
			expectedMismatches: []ast.Annotated{
				seltest.Annotated(map[string]string{
					policyhierarchy.ClusterSelectorAnnotationKey: "sel-1",
				}),
				seltest.Annotated(map[string]string{
					policyhierarchy.ClusterSelectorAnnotationKey: "sel-2",
				}),
				seltest.Annotated(map[string]string{
					policyhierarchy.ClusterSelectorAnnotationKey: "unknown-selector",
				}),
			},
		},
		{
			name: "Unlabeled cluster matches any selector",
			clusters: []clusterregistry.Cluster{
				seltest.Cluster("cluster-1", map[string]string{}),
			},
			selectors: []policyhierarchy.ClusterSelector{
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
				selectors:   map[string]policyhierarchy.ClusterSelector{},
			},
			expectedForEach: map[string]policyhierarchy.ClusterSelector{
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
