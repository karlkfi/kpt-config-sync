// Package metadata provides validation checks for errors in Resource metadata
//
// These checks are specifically on the `metadata` fields, which are defined for all Resources.
// Validators MAY be triggered by Group/Version/Kind, but SHOULD NOT access Kind-specific fields.
// Kinds with validation specific to their fields SHOULD have their own dedicated package.
package metadata
