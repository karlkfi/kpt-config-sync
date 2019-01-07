package veterrors

import "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"

// IllegalKindInNamespacesErrorCode is the error code for IllegalKindInNamespacesError
const IllegalKindInNamespacesErrorCode = "1038"

func init() {
	register(IllegalKindInNamespacesErrorCode, nil, "")
}

// IllegalKindInNamespacesError reports that an object has been illegally defined in namespaces/
type IllegalKindInNamespacesError struct {
	ResourceID
}

// Error implements error
func (e IllegalKindInNamespacesError) Error() string {
	return format(e,
		"Resources of the below Kind may not be declared in %[2]s/:\n\n"+
			"%[1]s",
		printResourceID(e), repo.NamespacesDir)
}

// Code implements Error
func (e IllegalKindInNamespacesError) Code() string {
	return IllegalKindInNamespacesErrorCode
}
