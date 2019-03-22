// Package semantic package provides validation checks for semantic errors in Nomos resource
// directories.
//
// For the purpose of this package, we define a "semantic error" to be a configuration error which
// cannot be determined by looking at a single Resource. Examples of semantic errors include
// detecting duplicate directories and verifying that a NamespaceSelector references a Namespace
// that exists.
package semantic
