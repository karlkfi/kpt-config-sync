package validate

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/yaml"
)

// RootSync checks if the given FileObject is a RootSync and if so, verifies
// that its fields are valid.
func RootSync(obj ast.FileObject) status.Error {
	if obj.GetObjectKind().GroupVersionKind().GroupKind() != kinds.RootSyncV1Beta1().GroupKind() {
		return nil
	}
	s, err := obj.Structured()
	if err != nil {
		return err
	}
	var rs *v1beta1.RootSync
	if obj.GroupVersionKind() == kinds.RootSyncV1Alpha1() {
		rs, err = toRootSyncV1Beta1(s.(*v1alpha1.RootSync))
		if err != nil {
			return err
		}
	} else {
		rs = s.(*v1beta1.RootSync)
	}
	return GitSpec(rs.Spec.Git, rs)
}

func toRootSyncV1Beta1(rs *v1alpha1.RootSync) (*v1beta1.RootSync, status.Error) {
	data, err := yaml.Marshal(rs)
	if err != nil {
		return nil, status.ResourceWrap(err, "failed marshalling", rs)
	}
	s := &v1beta1.RootSync{}
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, status.ResourceWrap(err, "failed to convert to v1beta1 RootSync", rs)
	}
	return s, nil
}
