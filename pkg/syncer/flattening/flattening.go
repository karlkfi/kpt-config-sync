// Flattening contains the logic used to flatten the hierarchical policy
// representation for policies to which flattening applies.  See tests for
// usage examples.
//
// TODO(fmil): Optimize this a bit.
// TODO(fmil): PolicyNode is not thread safe.  Handle that perhaps once it is
// used in the syncer process.
package flattening

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	api "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/util/set/stringset"
	rbac "k8s.io/api/rbac/v1"
)

const (
	// NoParent is a special label denoting that the node is not intended to
	// have a parent.
	NoParent Parent = Parent(api.NoParentNamespace)
)

type Name string                        // Name of a node.
type Parent string                      // Parent is the name of the parent node.
type Policy interface{}                 // Policy is the generic stored policy.
type PolicyNodeSet map[*PolicyNode]bool // A policy node set.

// PolicyNode is a node in the tree of custom policies.
type PolicyNode struct {
	// If set to nil, the parent is not known yet.  If set to self, it's the
	// root node.
	parent *PolicyNode
	// The set of children of this policy node.
	children PolicyNodeSet
	// This is the policy attached to this policy node.  As policy node may be
	// created out of order, it is possible that this is unset.
	policy Policy
}

// NewPolicyNode creates a policy node with the given parent and policy content.
// Both parameters may be nil, in which case a "dummy" policy node is created.
func NewPolicyNode(parent *PolicyNode, policy Policy) *PolicyNode {
	return &PolicyNode{parent, map[*PolicyNode]bool{}, policy}
}

// Add adds child as a child to this policy node.  It reparents the child if
// needed.
func (n *PolicyNode) Add(child *PolicyNode) {
	if n == nil {
		// Skip the operation for the root node.
		return
	}
	n.children[child] = true
	if child.parent != n {
		child.parent.Remove(child)
	}
	child.parent = n
}

// Set sets policy on this policy node.  For example, when the policy node is
// a dummy node that has not been filled in yet.
func (n *PolicyNode) Set(policy Policy) {
	if policy == nil {
		panic("policy may not be nil.")
	}
	n.policy = policy
}

// Remove removes the child policy node from this policy node.  Nothing is
// changed if the child isn't actually a descendant of this policy node.
func (n *PolicyNode) Remove(child *PolicyNode) error {
	if n == nil {
		// Skip the operation for the root node.
		return nil
	}

	if _, ok := n.children[child]; !ok {
		return fmt.Errorf("child not found: %v in parent: %v", child, n)
	}
	delete(n.children, child)
	child.parent = nil
	return nil
}

// PolicyTree contains a generic tree of policies.
type PolicyTree struct {
	// nodesByName contains a mapping from a node name to the corresponding
	// policy node.
	nodesByName map[Name]*PolicyNode
}

// NewPolicyTree creates an empty policy tree.
func NewPolicyTree() *PolicyTree {
	return &PolicyTree{map[Name]*PolicyNode{}}
}

// Lookup looks up a policy node by name in the given policy tree.  Returns the
// found policy node, or nil if the policy node was not found.
func (t *PolicyTree) Lookup(name Name) *PolicyNode {
	if result, ok := t.nodesByName[name]; !ok {
		glog.V(9).Infof("not found: %v", name)
		return nil
	} else {
		glog.V(9).Infof("found: node[%v]=%#v", name, result)
		return result
	}
}

// Add inserts the given policy node into the tree under the given name.
func (t *PolicyTree) Add(name Name, node *PolicyNode) {
	t.nodesByName[name] = node
}

// Upsert inserts or updates a policy by a given 'name' and hangs it off of the
// specified parent.  The parent policy may not exist at the time Upsert is
// called.
func (t *PolicyTree) Upsert(name Name, parent Parent, policy Policy) {
	// Get an existing parent,  or create a dummy parent.
	parentNode := t.Lookup(Name(parent))
	if parentNode == nil {
		// Parent is either not defined yet, or shouldn't exist altogether.
		if parent != NoParent {
			// The parent isn't defined yet, but it should exist.
			parentNode = NewPolicyNode(nil, nil)
			t.Add(Name(parent), parentNode)
		}
	}
	if parentNode == nil && parent != NoParent {
		panic(fmt.Sprintf("parentNode must not be nil: %v, %v", name, parent))
	}
	policyNode := t.Lookup(name)
	if policyNode == nil {
		// There is no policy node yet by this name.
		policyNode = NewPolicyNode(parentNode, policy)
	}
	// Update the policy content of the node.
	policyNode.Set(policy)
	parentNode.Add(policyNode)
	t.Add(name, policyNode)
	glog.V(10).Infof("nodesByName after %v:\n%v", name, spew.Sdump(t.nodesByName))
}

// FlattenPolicyFunc is used to flatten the generic policy of a parent and a
// child policy node.  The exact method for doing so depends on the concrete
// type of the policy involved.  Used in PolicyNode.Eval.
type FlattenPolicyFunc func(parent, child Policy) Policy

// Eval evaluates the aggregate value of the policy on this node.
func (n *PolicyNode) Eval(flatten FlattenPolicyFunc) Policy {
	if flatten == nil {
		panic("flatten must not be nil")
	}
	if n == nil {
		return nil // Nonexistent node.
	}
	result := n.policy
	for ; n.parent != nil; n = n.parent {
		if n.parent.policy != nil {
			result = flatten(n.parent.policy, result)
		}
	}
	return result
}

// FlatenRoleBindings flattens the RoleBinding RBAC policies.  RoleBinding
// policies are flattened by joining the parent and the child RoleBindings,
// parent appearing first.  The input policies are *[]rbac.RoleBinding, and the
// returned policy is *[]rbac.RoleBinding as well.
func FlattenRoleBindings(parent, child Policy) Policy {
	// TODO(fmil): This and the role flattening will need to fix the
	// resulting namespace of roles and role bindings in the flattened
	// result.
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}
	ra, okA := parent.(*[]rbac.RoleBinding)
	rb, okB := child.(*[]rbac.RoleBinding)
	if !okA || !okB {
		panic("type mismatch")
	}
	if _, ok := parent.(*[]rbac.RoleBinding); !ok {
		panic(fmt.Sprintf("type mismatch: parent:%#v, child%#v", parent, child))
	}
	result := append(*ra, *rb...)
	// TODO(fmil): Namespace conversion?
	return &result
}

// FlatenRoles flattens the Role RBAC policies.  Role definitions are flattened
// by concatenating uniquely named roles.  If role names conflict, the role
// definition from the parent wins over the child role by the same name.
// parent and child values are *[]rbac.Role, and the return value is
// *[]rbac.Role.
func FlattenRoles(parent, child Policy) Policy {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}
	parentRoles, okParent := parent.(*[]rbac.Role)
	childRoles, okChild := child.(*[]rbac.Role)
	if !okParent || !okChild {
		panic(fmt.Sprintf("type mismatch: parent:%#v, child%#v", parent, child))
	}
	result := []rbac.Role{}
	seenName := stringset.New()
	for _, roleSet := range [][]rbac.Role{*parentRoles, *childRoles} {
		for _, role := range roleSet {
			if seenName.Contains(role.Name) {
				continue
			}
			seenName.Add(role.Name)
			result = append(result, *role.DeepCopy())
			// TODO(fmil): Namespace conversion?
		}
	}
	return &result
}
