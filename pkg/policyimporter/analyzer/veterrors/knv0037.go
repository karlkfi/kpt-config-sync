package veterrors

import "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"

// IllegalKindInClusterregistryError reports that an object has been illegally defined in clusterregistry/
type IllegalKindInClusterregistryError struct {
	ResourceID
}

// Error implements error
func (e IllegalKindInClusterregistryError) Error() string {
	return format(e,
		"Resources of the below Kind may not be declared in %[2]s/:\n\n"+
			"%[1]s",
		printResourceID(e), repo.ClusterRegistryDir)
}

// Code implements Error
func (e IllegalKindInClusterregistryError) Code() string {
	return IllegalKindInClusterregistryErrorCode
}
