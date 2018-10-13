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

package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	policyhierarchy "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type clusterselectorsTestCase struct {
	name            string
	clusters        []clusterregistry.Cluster
	selectors       []policyhierarchy.ClusterSelector
	expectedMapping ClusterSelectors
	expectedForEach map[string]policyhierarchy.ClusterSelector
}

func Opts() cmp.Options {
	return cmp.Options{
		cmp.AllowUnexported(ClusterSelectors{}),
	}
}

func (tc *clusterselectorsTestCase) run(t *testing.T) {
	s, err := NewClusterSelectors(tc.clusters, tc.selectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(tc.expectedMapping, *s, Opts()...) {
		t.Errorf("GetClusterSelectors(_)=%v, want: %v\ndiff:\n%v", *s, tc.expectedMapping, cmp.Diff(tc.expectedMapping, *s, Opts()))
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
				selectorToClusters: map[string]clusterSet{},
				clusters:           map[string]clusterregistry.Cluster{},
				selectors:          map[string]policyhierarchy.ClusterSelector{},
			},
		},
		{
			name:      "Only selector list",
			selectors: []policyhierarchy.ClusterSelector{},
			expectedMapping: ClusterSelectors{
				clusters:           map[string]clusterregistry.Cluster{},
				selectors:          map[string]policyhierarchy.ClusterSelector{},
				selectorToClusters: map[string]clusterSet{},
			},
		},
		{
			name: "Basic",
			clusters: []clusterregistry.Cluster{
				cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
			},
			selectors: []policyhierarchy.ClusterSelector{
				selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "prod",
						},
					}),
			},
			expectedMapping: ClusterSelectors{
				selectors: map[string]policyhierarchy.ClusterSelector{
					"sel-1": selector("sel-1",
						metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "prod",
							},
						}),
				},
				clusters: map[string]clusterregistry.Cluster{
					"cluster-1": cluster("cluster-1", map[string]string{
						"env": "prod",
					}),
				},
				selectorToClusters: map[string]clusterSet{
					"sel-1": clusterSet{
						"cluster-1": true,
					},
				},
			},
		},
		{
			name: "Mismatching labels",
			clusters: []clusterregistry.Cluster{
				cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
			},
			selectors: []policyhierarchy.ClusterSelector{
				selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "test",
						},
					}),
			},
			expectedMapping: ClusterSelectors{
				clusters: map[string]clusterregistry.Cluster{
					"cluster-1": cluster("cluster-1", map[string]string{
						"env": "prod",
					}),
				},
				selectors: map[string]policyhierarchy.ClusterSelector{
					"sel-1": selector("sel-1",
						metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "test",
							},
						}),
				},
				selectorToClusters: map[string]clusterSet{},
			},
		},
		{
			name: "Unlabeled cluster matches any selector",
			clusters: []clusterregistry.Cluster{
				cluster("cluster-1", map[string]string{}),
			},
			selectors: []policyhierarchy.ClusterSelector{
				selector("sel-1",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "test",
						},
					}),
			},
			expectedMapping: ClusterSelectors{
				selectorToClusters: map[string]clusterSet{},
				clusters: map[string]clusterregistry.Cluster{
					"cluster-1": cluster("cluster-1", map[string]string{}),
				},
				selectors: map[string]policyhierarchy.ClusterSelector{
					"sel-1": selector("sel-1",
						metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "test",
							},
						}),
				},
			},
			expectedForEach: map[string]policyhierarchy.ClusterSelector{
				"sel-1": selector("sel-1",
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

func cluster(name string, labels map[string]string) clusterregistry.Cluster {
	return clusterregistry.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func selector(name string, selector metav1.LabelSelector) policyhierarchy.ClusterSelector {
	return policyhierarchy.ClusterSelector{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy.ClusterSelectorSpec{
			Selector: selector,
		},
	}
}
