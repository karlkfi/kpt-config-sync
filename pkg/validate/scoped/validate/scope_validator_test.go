package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestClusterScoped(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Object without metadata.namespace passes",
			obj:  fake.ClusterRole(),
		},
		{
			name:    "Object with metadata.namespace fails",
			obj:     fake.ClusterRole(core.Namespace("hello")),
			wantErr: nonhierarchical.IllegalNamespaceOnClusterScopedResourceError(fake.ClusterRole()),
		},
		{
			name: "Object with namespace selector fails",
			obj: fake.ClusterRole(
				core.Annotation(metadata.NamespaceSelectorAnnotationKey, "value")),
			wantErr: nonhierarchical.IllegalNamespaceSelectorAnnotationError(fake.ClusterRole()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ClusterScoped(tc.obj)
			if !errors.Is(errs, tc.wantErr) {
				t.Errorf("got ClusterScoped() error %v, want %v", errs, tc.wantErr)
			}
		})
	}
}

func TestClusterScopedForNamespaceReconciler(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name:    "Cluster scoped object fails",
			obj:     fake.ClusterRole(),
			wantErr: shouldBeInRootErr(fake.ClusterRole()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ClusterScopedForNamespaceReconciler(tc.obj)
			if !errors.Is(errs, tc.wantErr) {
				t.Errorf("got ClusterScopedForNamespaceReconciler() error %v, want %v", errs, tc.wantErr)
			}
		})
	}
}

func TestNamespaceScoped(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Object without metadata.namespace passes",
			obj:  fake.Role(),
		},
		{
			name: "Object with metadata.namespace passes",
			obj:  fake.Role(core.Namespace("hello")),
		},
		{
			name: "Object with namespace selector passes",
			obj: fake.Role(
				core.Annotation(metadata.NamespaceSelectorAnnotationKey, "value")),
		},
		{
			name: "Object with namespace and namespace selector fails",
			obj: fake.Role(
				core.Namespace("hello"),
				core.Annotation(metadata.NamespaceSelectorAnnotationKey, "value")),
			wantErr: nonhierarchical.NamespaceAndSelectorResourceError(fake.Role()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := NamespaceScoped(tc.obj)
			if !errors.Is(errs, tc.wantErr) {
				t.Errorf("got NamespaceScoped() error %v, want %v", errs, tc.wantErr)
			}
		})
	}
}
