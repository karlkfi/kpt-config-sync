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

package filesystem

import (
	"testing"
	"time"

	"k8s.io/client-go/restmapper"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	nomosfs "github.com/google/nomos/pkg/policyimporter/filesystem"
	nomostesting "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	aNamespaceConfig = `
apiVersion: v1
kind: Namespace
metadata:
  name: bar
`
	aProjectSync = `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: project
spec:
  groups:
  - group: bespin.dev
    kinds:
    - kind: Project
      versions:
      - version: v1
`

	aProjectConfig = `
apiVersion: bespin.dev/v1
kind: Project
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: project-sample
spec:
  # Add fields here
  foo: bar
`

	aRepo = `
kind: Repo
apiVersion: nomos.dev/v1alpha1
spec:
  version: "0.1.0"
metadata:
  name: repo
`
)

func TestBespinParser(t *testing.T) {
	var tests = []struct {
		name, root      string
		files           nomostesting.FileContentMap
		wantPolicyCount map[string]int
		wantErr         bool
	}{
		{
			name: v1.ProjectKind,
			root: "foo",
			files: nomostesting.FileContentMap{
				"system/nomos.yaml":          aRepo,
				"system/project.yaml":        aProjectSync,
				"hierarchy/bar/ns.yaml":      aNamespaceConfig,
				"hierarchy/bar/project.yaml": aProjectConfig,
			},
			wantPolicyCount: map[string]int{nomosv1.RootPolicyNodeName: 0, "bar": 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Skip("b/123720068")
			d := nomostesting.NewTestDir(t, tc.root)
			defer d.Remove()

			for k, v := range tc.files {
				d.CreateTestFile(k, v)
			}

			f := nomostesting.NewTestFactory(t, &restmapper.APIGroupResources{
				Group: metav1.APIGroup{
					Name:             "bespin.dev",
					Versions:         []metav1.GroupVersionForDiscovery{{Version: "v1"}},
					PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
				},
				VersionedResources: map[string][]metav1.APIResource{
					"v1": {
						{Name: "projects", Namespaced: true, Kind: v1.ProjectKind},
					},
				},
			})
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			p, err := nomosfs.NewParserWithFactory(f, nomosfs.ParserOpt{Validate: true, Extension: &BespinVisitorProvider{}})
			if err != nil {
				t.Fatal(err)
			}

			policies, err := p.Parse(d.Root(), "", time.Time{})
			if (err != nil) != tc.wantErr {
				t.Errorf("got error = %v, want error %v", err, tc.wantErr)
			}

			if len(tc.wantPolicyCount) > 0 {
				if policies == nil {
					t.Fatal(err)
				}

				n := make(map[string]int)
				for k, v := range policies.PolicyNodes {
					n[k] = 0
					for _, res := range v.Spec.Resources {
						for _, version := range res.Versions {
							n[k] += len(version.Objects)
						}
					}
				}
				if diff := cmp.Diff(n, tc.wantPolicyCount); diff != "" {
					t.Errorf("Actual and expected number of policy nodes didn't match: %v", diff)
				}
			}
		})
	}
}
