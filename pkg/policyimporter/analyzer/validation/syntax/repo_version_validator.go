package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/runtime"
)

// AllowedRepoVersion is the only allowed Repo version.
const AllowedRepoVersion = "0.1.0"

// RepoVersionValidator validates that
var RepoVersionValidator = &ObjectValidator{
	validate: func(source string, object runtime.Object) error {
		switch o := object.(type) {
		case *v1alpha1.Repo:
			if version := o.Spec.Version; version != AllowedRepoVersion {
				return vet.UnsupportedRepoSpecVersion{Source: source, Name: o.Name, Version: version}
			}
		}
		return nil
	},
}
