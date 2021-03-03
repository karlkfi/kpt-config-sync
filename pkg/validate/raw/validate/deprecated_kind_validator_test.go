package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/api/extensions/v1beta1"
)

func TestDeprecatedKinds(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Non-deprecated Deployment",
			obj:  fake.Deployment("namespaces/foo"),
		},
		{
			name:    "Deprecated Deployment",
			obj:     fake.Unstructured(v1beta1.SchemeGroupVersion.WithKind("Deployment")),
			wantErr: fake.Error(nonhierarchical.DeprecatedGroupKindErrorCode),
		},
		{
			name:    "Deprecated PodSecurityPolicy",
			obj:     fake.Unstructured(v1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy")),
			wantErr: fake.Error(nonhierarchical.DeprecatedGroupKindErrorCode),
		},
		{
			name:    "Deprecated Ingress",
			obj:     fake.Unstructured(v1beta1.SchemeGroupVersion.WithKind("Ingress")),
			wantErr: fake.Error(nonhierarchical.DeprecatedGroupKindErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := DeprecatedKinds(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got DeprecatedKinds() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
