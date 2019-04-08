package id

import (
	"fmt"
)

// TreeNode represents a named node in the policy hierarchy.
type TreeNode interface {
	// Path is the embedded interface providing path information to this node.
	Path
	// Name returns the name of this node.
	Name() string
}

// PrintTreeNode returns a convenient representation of a TreeNode for error messages.
func PrintTreeNode(n TreeNode) string {
	return fmt.Sprintf("path: %[1]s\n"+
		"name: %[2]s",
		n.SlashPath(), n.Name())
}
