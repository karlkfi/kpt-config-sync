package veterrors

// InvalidMetadataNameError represents the usage of a non-RFC1123 compliant metadata.name
type InvalidMetadataNameError struct {
	ResourceID
}

// Error implements error.
func (e InvalidMetadataNameError) Error() string {
	return format(e,
		"Resources MUST define a metadata.name which is a valid RFC1123 DNS subdomain. Rename or remove the Resource:\n\n"+
			"%[1]s",
		printResourceID(e))
}

// Code implements Error
func (e InvalidMetadataNameError) Code() string { return InvalidMetadataNameErrorCode }
