package id

// DisallowedField is the json path of a field that is not allowed in imported objects.
type DisallowedField string

const (
	// OwnerReference represents the ownerReference field in a metav1.Object.
	OwnerReference DisallowedField = "metadata.ownerReference"
	// SelfLink represents the selfLink field in a metav1.Object.
	SelfLink DisallowedField = "metadata.selfLink"
	// UID represents the uid field in a metav1.Object.
	UID DisallowedField = "metadata.uid"
	// ResourceVersion represents the resourceVersion field in a metav1.Object.
	ResourceVersion DisallowedField = "metadata.resourceVersion"
	// Generation represents the generation field in a metav1.Object.
	Generation DisallowedField = "metadata.generation"
	// CreationTimestamp represents the creationTimestamp field in a metav1.Object.
	CreationTimestamp DisallowedField = "metadata.creationTimestamp"
	// DeletionTimestamp represents the deletionTimestamp field in a metav1.Object.
	DeletionTimestamp DisallowedField = "metadata.deletionTimestamp"
	// DeletionGracePeriodSeconds represents the deletionGracePeriodSeconds field in a metav1.Object.
	DeletionGracePeriodSeconds DisallowedField = "metadata.deletionGracePeriodSeconds"
	// Status represents the status field in a runtime.Object.
	Status DisallowedField = "status"
)
