package gcpconfig

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"

	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	nomoskinds "github.com/google/nomos/pkg/kinds"
)

func TestFileNameValidator(t *testing.T) {
	var tests = []struct {
		name    string
		object  ast.FileObject
		wantErr bool
	}{
		{
			name:   "Organization with correct filename",
			object: orgFileObj("hierarchy/foo/gcp-organization.yaml"),
		},
		{
			name:    "Organization with incorrect filename",
			object:  orgFileObj("hierarchy/foo/bad.yaml"),
			wantErr: true,
		},
		{
			name:    "Organization with incorrect filename (case sensitive)",
			object:  orgFileObj("hierarchy/foo/Gcp-Organization.yaml"),
			wantErr: true,
		},
		{
			name:   "Folder with correct filename",
			object: folderFileObj("hierarchy/foo/gcp-folder.yaml"),
		},
		{
			name:    "Folder with incorrect filename",
			object:  folderFileObj("hierarchy/foo/bad.yaml"),
			wantErr: true,
		},
		{
			name:    "Folder with incorrect filename (case sensitive)",
			object:  folderFileObj("hierarchy/foo/Gcp-Folder.yaml"),
			wantErr: true,
		},
		{
			name:   "Project with correct filename",
			object: projectFileObj("hierarchy/foo/gcp-project.yaml"),
		},
		{
			name:    "Project with incorrect filename",
			object:  projectFileObj("hierarchy/foo/bad.yaml"),
			wantErr: true,
		},
		{
			name:    "Project with incorrect filename (case sensitive)",
			object:  projectFileObj("hierarchy/foo/Gcp-Project.yaml"),
			wantErr: true,
		},
		{
			name:   "IAMPolicy with correct filename",
			object: iamPolicyFileObj("hierarchy/foo/bar/gcp-iam-policy.yaml"),
		},
		{
			name:    "IAMPolicy with incorrect filename (case sensitive)",
			object:  iamPolicyFileObj("hierarchy/foo/bar/Gcp-Iam-policy.yaml"),
			wantErr: true,
		},
		{
			name:    "IAMPolicy with incorrect filename",
			object:  iamPolicyFileObj("hierarchy/foo/bar/bad.yaml"),
			wantErr: true,
		},
		{
			name:   "OrganizationPolicy with correct filename",
			object: orgPolicyFileObj("hierarchy/foo/bar/gcp-organization-policy.yaml"),
		},
		{
			name:    "OrganizationPolicy with incorrect filename",
			object:  orgPolicyFileObj("hierarchy/foo/bar/bad.yaml"),
			wantErr: true,
		},
		{
			name:    "OrganizationPolicy with incorrect filename (case sensitive)",
			object:  orgPolicyFileObj("hierarchy/foo/bar/Gcp-Organization-policy.yaml"),
			wantErr: true,
		},
		{
			name: "non-GCP resource",
			object: asttesting.NewFakeFileObject(
				nomoskinds.Role(),
				"hierarchy/foo/bar.yaml"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := treetesting.BuildTree(t, tc.object)

			v := NewFilenameValidator()
			root.Accept(v)

			switch {
			case tc.wantErr && v.Error() == nil:
				t.Fatalf("got nil, want error")
			case !tc.wantErr && v.Error() != nil:
				t.Fatalf("got %v, want nil", v.Error())
			}
		})
	}
}
