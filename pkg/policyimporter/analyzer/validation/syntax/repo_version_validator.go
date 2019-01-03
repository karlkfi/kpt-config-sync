package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

// AllowedRepoVersion is the only allowed Repo version.
const AllowedRepoVersion = "0.1.0"

// RepoVersionValidator validates that
var RepoVersionValidator = &FileObjectValidator{
	ValidateFn: func(object ast.FileObject) error {
		switch o := object.Object.(type) {
		case *v1alpha1.Repo:
			if version := o.Spec.Version; version != AllowedRepoVersion {
				return vet.UnsupportedRepoSpecVersion{ResourceID: &object, Version: version}
			}
		}
		return nil
	},
}
