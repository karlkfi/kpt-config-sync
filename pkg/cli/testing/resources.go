package testing

import (
	policyhierarchy "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	apicore "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// NamespaceTypeMeta is the type meta for a namespace
	NamespaceTypeMeta = meta.TypeMeta{
		Kind: "Namespace",
	}
)

// NewNamespace creates a Namespace object named 'name', with
// Nomos-style parent 'parent'.
// TODO(filmil): Find places that could use this function and plug it in.
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
