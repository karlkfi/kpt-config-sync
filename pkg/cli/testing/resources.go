package testing

import (
	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	apicore "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	NamespaceTypeMeta = meta.TypeMeta{
		Kind: "Namespace",
	}
)

// namespace creates a Namespace object named 'name', with
// Stolos-style parent 'parent'.
// TODO(fmil): Find places that could use this function and plug it in.
func NewNamespace(name, parent string) *apicore.Namespace {
	return &apicore.Namespace{
		TypeMeta: NamespaceTypeMeta,
		ObjectMeta: meta.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				policyhierarchy.ParentLabelKey: parent,
			},
		},
	}
}
