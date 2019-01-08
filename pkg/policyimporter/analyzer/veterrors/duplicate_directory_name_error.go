package veterrors

// TODO(b/77981474) Remove this error.

import (
	"sort"
	"strings"
)

// DuplicateDirectoryNameErrorCode is the error code for DuplicateDirectoryNameError
const DuplicateDirectoryNameErrorCode = "1002"

var duplicateDirectoryNameErrorExamples = []Error{
	DuplicateDirectoryNameError{
		Duplicates: []string{"foo/bar", "qux/bar"},
	},
	DuplicateDirectoryNameError{
		Duplicates: []string{"bar", "bar/foo/bar"},
	},
}

var duplicateDirectoryNameErrorExplanation = `
The name of every directory in a repository MUST be distinct within the entire repo.

To fix, rename one of the conflicting directories.
If the renamed directory contains a Namespace, you also need to update {{.Q "metadata.name"}} to reflect the new directory name.

# Examples

This can happen if two directories with different paths share the same name.
For instance, a directory structure that includes both foo/bar and baz/bar generates this error.

{{.CodeMode}}
namespaces/
├── foo/
│   └── bar/
└── qux/
    └── bar/
{{.CodeMode}}

The above would produce this error:

{{.CodeMode}}
{{index .Examples 0}}
{{.CodeMode}}

Another way to cause is error is a directory structure such as foo/foo/ or foo/bar/foo/.

{{.CodeMode}}
namespaces/
└── bar/
    └── foo/
        └── bar/
{{.CodeMode}}

The above would produce this error:

{{.CodeMode}}
{{index .Examples 1}}
{{.CodeMode}}
`

func init() {
	register(DuplicateDirectoryNameErrorCode, duplicateDirectoryNameErrorExamples, duplicateDirectoryNameErrorExplanation)
}

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
type DuplicateDirectoryNameError struct {
	Duplicates []string
}

// Error implements error.
func (e DuplicateDirectoryNameError) Error() string {
	// Ensure deterministic node printing order.
	sort.Strings(e.Duplicates)
	return format(e,
		"Directory names MUST be unique. Rename one of these directories:\n\n"+
			"%[1]s",
		strings.Join(e.Duplicates, "\n"))
}

// Code implements Error
func (e DuplicateDirectoryNameError) Code() string { return DuplicateDirectoryNameErrorCode }
