package id

// DisallowedField is the json path of a field that is not allowed in imported objects.
type DisallowedField string

// OwnerReference represents the ownerReference field in a metav1.Object.
const OwnerReference DisallowedField = "metadata.ownerReference"
