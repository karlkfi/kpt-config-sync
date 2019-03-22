// Package syntax package provides validation checks for syntax errors in Nomos resource
// directories.
//
// For the purpose of this package, we define a "syntax error" to be a configuration error which
// can be determined by looking at a single Resource. Examples of syntax errors include invalid
// names or missing required properties. This is as opposed to "semantic errors", which would
// include errors such as detecting that two Resources of the same Kind share the same name.
package syntax
