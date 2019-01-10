package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// resourceMeta is a minimal implementation of ResourceMeta for use in tests.
type resourceMeta struct {
	source           string
	name             string
	groupVersionKind schema.GroupVersionKind
	meta             metav1.Object
}

var _ ResourceMeta = resourceMeta{}

// RelativeSlashPath implements ResourceMeta
func (m resourceMeta) RelativeSlashPath() string { return m.source }

// Dir implements ResourceMeta
func (m resourceMeta) Dir() nomospath.Relative { return nomospath.NewFakeRelative(m.source).Dir() }

// Name implements ResourceMeta
func (m resourceMeta) Name() string { return m.name }

// GroupVersionKind implements ResourceMeta
func (m resourceMeta) GroupVersionKind() schema.GroupVersionKind { return m.groupVersionKind }

// MetaObject implements ResourceMeta
func (m resourceMeta) MetaObject() metav1.Object { return m.meta }
