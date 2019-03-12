package id

import "github.com/google/nomos/pkg/status"

// ResourceError defines a status error related to one or more k8s resources.
type ResourceError interface {
	status.Error
	Resources() []Resource
}
