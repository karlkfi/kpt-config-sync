package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestIllegalCRD(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Anvil v1beta1 CRD",
			obj:  crdv1beta1("crd", kinds.Anvil()),
		},
		{
			name:    "ClusterConfig v1beta1 CRD",
			obj:     crdv1beta1("crd", kinds.ClusterConfig()),
			wantErr: fake.Error(nonhierarchical.UnsupportedObjectErrorCode),
		},
		{
			name:    "ClusterConfig v1 CRD",
			obj:     crdv1("crd", kinds.ClusterConfig()),
			wantErr: fake.Error(nonhierarchical.UnsupportedObjectErrorCode),
		},
		{
			name:    "RepoSync v1beta1 CRD",
			obj:     crdv1beta1("crd", kinds.RepoSyncV1Beta1()),
			wantErr: fake.Error(nonhierarchical.UnsupportedObjectErrorCode),
		},
		{
			name:    "RepoSync v1 CRD",
			obj:     crdv1("crd", kinds.RepoSyncV1Beta1()),
			wantErr: fake.Error(nonhierarchical.UnsupportedObjectErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := IllegalCRD(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got IllegalCRD() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
