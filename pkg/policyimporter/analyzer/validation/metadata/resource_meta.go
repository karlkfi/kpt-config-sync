package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceMeta provides a Resource's identifier and its metadata.
type ResourceMeta interface {
	id.Resource
	MetaObject() metav1.Object
}
