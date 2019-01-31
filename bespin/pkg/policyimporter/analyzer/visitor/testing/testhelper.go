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

package testing

import (
	"time"

	v1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// ImportToken defines a default token to use for testing.
	ImportToken = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	// ImportTime defines a default time to use for testing.
	ImportTime = time.Date(2017, 8, 10, 5, 16, 00, 0, time.FixedZone("PDT", -7*60*60))
)

// Helper provides a number of pre-built types for use in testcases.  This does not set an ImportToken
// or ImportTime
var Helper TestHelper

// TestHelper provides a number of pre-built types for use in testcases.
type TestHelper struct {
	ImportToken string
	ImportTime  time.Time
}

// NewTestHelper returns a TestHelper with default import token and time.
func NewTestHelper() *TestHelper {
	return &TestHelper{
		ImportToken: ImportToken,
		ImportTime:  ImportTime,
	}
}

// GCPProject returns a new, sample GCP project object.
func (t *TestHelper) GCPProject(name string) *v1.Project {
	return &v1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.ProjectKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.ProjectSpec{
			DisplayName: name,
		},
	}
}

// GCPFolder returns a new, sample GCP folder object.
func (t *TestHelper) GCPFolder(name string) *v1.Folder {
	return &v1.Folder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.FolderKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.FolderSpec{
			DisplayName: name,
		},
	}
}

// GCPOrg returns a new, sample GCP organization object.
func (t *TestHelper) GCPOrg(name string) *v1.Organization {
	return &v1.Organization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.OrganizationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.OrganizationSpec{
			ID: 123456789,
		},
	}
}

// GCPIAMPolicy returns a new, sample GCP IAMPolicy object.
func (t *TestHelper) GCPIAMPolicy(name string) *v1.IAMPolicy {
	return &v1.IAMPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IAMPolicyKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.IAMPolicySpec{
			Bindings: []v1.IAMPolicyBinding{},
		},
	}
}

// GCPOrgPolicy returns a new, sample GCP OrganizationPolicy object.
func (t *TestHelper) GCPOrgPolicy(name string) *v1.OrganizationPolicy {
	return &v1.OrganizationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.OrganizationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.OrganizationPolicySpec{
			Constraints: []v1.OrganizationPolicyConstraint{},
		},
	}
}
