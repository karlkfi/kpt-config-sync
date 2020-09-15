package client

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// ResourceConflictCode is the code for API Server errors resulting from a
// mismatch between our cached set of objects and the cluster's..
const ResourceConflictCode = "2008"

var retriableConflictBuilder = status.NewErrorBuilder(ResourceConflictCode)

// ConflictCreateAlreadyExists means we tried to create an object which already
// exists.
func ConflictCreateAlreadyExists(err error, resource id.Resource) status.Error {
	return retriableConflictBuilder.
		Wrap(err).
		Sprint("tried to create resource that already exists").
		BuildWithResources(resource)
}

// ConflictUpdateDoesNotExist means we tried to update an object which does not
// exist.
func ConflictUpdateDoesNotExist(err error, resource id.Resource) status.Error {
	return retriableConflictBuilder.
		Wrap(err).
		Sprint("tried to update resource which does not exist").
		BuildWithResources(resource)
}

// ConflictUpdateOldVersion means we tried to update an object using an old
// version of the object.
func ConflictUpdateOldVersion(err error, resource id.Resource) status.Error {
	return retriableConflictBuilder.
		Wrap(err).
		Sprintf("tried to update with stale version of resource").
		BuildWithResources(resource)
}
