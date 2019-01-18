package k8s

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// nolint
func IsDeleted(objectMeta *v1.ObjectMeta) bool {
	return !objectMeta.DeletionTimestamp.IsZero()
}
