package flattening

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	api "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
)

const (
	none parentName = parentName(api.NoParentNamespace)
)

// Helper types for some additional type safety in the adjacency list below.
type nodeName string
type parentName string
type isPolicyspaceType bool

// TODO: Move the 4 maps below into one map.
type childrenSet map[nodeName]bool

// Mapping from node to parent name type.
type nodeToParentMap map[nodeName]parentName

// Mapping from node to whether it's a policyspace or not
type nodeToPolicyspaceMap map[nodeName]isPolicyspaceType

// Mapping from parent to a set of its children.
type parentToChildrenMap map[parentName]childrenSet

func (c *parentToChildrenMap) addChild(child nodeName, parent parentName) bool {
	this := *c
	if parent == none {
		return false
	}
	_, ok := this[parent]
	if !ok {
		// No children yet.
		this[parent] = childrenSet{}
	}
	children := this[parent]
	children[child] = true
	return true
}

func (c *parentToChildrenMap) removeChild(child nodeName, parent parentName) bool {
	if parent == none {
		return false
	}
	this := *c
	children, ok := this[parent]
	if !ok {
		return false
	}
	delete(children, child)
	return true
}

// PolicyTree is a tree stucture, with each node of type PolicyNode containing
// one generic Policy.
type PolicyTree struct {
	// nodesByName contains a mapping from a node name to the corresponding
	// policy node.
	nodesByName map[nodeName]Policy
	// isPolicyspaceMap contains a mapping from a node name to whether that node is a policyspace
	isPolicyspaceMap nodeToPolicyspaceMap
	// parentsMap is a mapping from a node name to its parent.  If a node has no
	// key in this map, then it has no parent.
	parentsMap nodeToParentMap
	// childrenMap is a mapping from a node name to the set of names of all its
	// children.
	childrenMap parentToChildrenMap
}

// NewPolicyTree creates an empty policy tree.
func NewPolicyTree() *PolicyTree {
	return &PolicyTree{map[nodeName]Policy{}, nodeToPolicyspaceMap{},
	nodeToParentMap{}, parentToChildrenMap{}}
}

// Lookup finds a policy node by name in the given policy tree.  Returns the
// found policy node, or error if the policy node is not found.
func (t *PolicyTree) Lookup(name string) (Policy, error) {
	policy, ok := t.nodesByName[nodeName(name)]
	if !ok {
		return Policy{}, fmt.Errorf("Policy not found: %q", name)
	}
	return policy, nil
}

// ParentOf returns the name of the parent of the named node.  If no parent
// exists (such as the case of a root node), returns a "not found" error.
func (t *PolicyTree) ParentOf(name string) (string, error) {
	parent, ok := t.parentsMap[nodeName(name)]
	if !ok {
		return string(none), fmt.Errorf("Parent not found: %q", name)
	}
	return string(parent), nil
}

// ParentOf returns the name of the parent of the named node.  If no parent
// exists (such as the case of a root node), returns a "not found" error.
func (t *PolicyTree) IsPolicyspace(name string) (bool, error) {
	isPolicyspace, ok := t.isPolicyspaceMap[nodeName(name)]
	if !ok {
		return false, fmt.Errorf("Policspace status not found: %q", name)
	}
	return bool(isPolicyspace), nil
}

// unparent removes the named node from its current parent.
func (t *PolicyTree) unparent(name nodeName) {
	if oldParent, ok := t.parentsMap[name]; ok {
		// There is an old parent.  Remove the child from that parent.
		t.childrenMap.removeChild(name, oldParent)
	}
}

// Upsert inserts or updates a policy by a given 'name' and hangs it off of the
// specified parent.  The parent policy may not have been defined yet at the
// time Upsert is called.
func (t *PolicyTree) Upsert(name string, parent string, isPolicyspace bool, policy Policy) {
	n := nodeName(name)
	p := parentName(parent)

	t.nodesByName[n] = policy
	t.isPolicyspaceMap[n] = isPolicyspaceType(isPolicyspace)
	t.unparent(n)
	t.childrenMap.addChild(n, p)
	if p != none {
		t.parentsMap[n] = p
	}
	if glog.V(10) {
		// Avoid materializing the long dump if no debug output was requested.
		glog.V(10).Infof("PolicyTree after Upsert(%q):\n%v", name, spew.Sdump(t))
	}
}

// Delete deletes the node with the given name from the tree.  This causes
// all current children to be deleted as well.
func (t *PolicyTree) Delete(name string) {
	n := nodeName(name)
	if children, ok := t.childrenMap[parentName(name)]; ok {
		for child := range children {
			t.Delete(string(child))
		}
	}
	delete(t.nodesByName, n)
	t.unparent(n)
	delete(t.parentsMap, n)
}

// Roots returns the names of all roots of the policy tree.  A "root" may be
// either the actual node on the top of the policy tree, or a node that isn't
// on the top, but whose parent node was removed from the policy tree due to
// some transient event.
func (t *PolicyTree) Roots() []string {
	result := []string{}
	for name := range t.nodesByName {
		if parent, ok := t.parentsMap[name]; !ok {
			// It's a root because there's no parent.
			result = append(result, string(name))
		} else {
			// It's a root because the parent is not yet there.
			if _, parentExists := t.nodesByName[nodeName(parent)]; !parentExists {
				result = append(result, string(name))
			}
		}
	}
	return result
}

func (t *PolicyTree) eval(name string) (Policy, error) {
	glog.V(10).Infof("eval(_,%q)", name)
	policy, _ := t.Lookup(name)
	parent, err := t.ParentOf(name)
	if err != nil {
		glog.V(10).Infof("parent not found for: %q: %v", name, err)
		return policy, nil
	}
	parentPolicy, err := t.eval(parent)
	if err != nil {
		return Policy{}, errors.Wrapf(err, "while evaluating: %q", name)
	}
	result := flatten(nodeName(name), parentPolicy, policy)
	if glog.V(10) {
		glog.V(10).Infof("eval(_,%q)=(%v,%v)", name, result, err)
	}
	return result, nil
}

// Eval evaluates the aggregate value of the policy on this node.  Returns
// the evaluated policy or an error.  It is an error to call Eval on a
// node that does not exist.
func (t *PolicyTree) Eval(name string) (Policy, error) {
	glog.V(10).Infof("Eval(_,%q)", name)
	_, err := t.Lookup(name)
	if err != nil {
		return Policy{}, errors.Wrapf(err, "while evaluating node: %q", name)
	}
	return t.eval(name)
}

// Visitor is used in the functions below.
type Visitor interface {
	Visit(name string, p *Policy)
}

// VisitSubtree visits all nodes in the subtree of node n, inclusive.  The
// Visitor.Visit method is called on each node seen on the way.
func (t *PolicyTree) VisitSubtree(node string, v Visitor) {
	children, ok := t.childrenMap[parentName(node)]
	if ok {
		for child := range children {
			t.VisitSubtree(string(child), v)
		}
	}
	policy, err := t.Lookup(node)
	if err != nil {
		glog.V(10).Infof("t.Lookup(%q)=%v", node, err)
		return
	}
	v.Visit(node, &policy)
}
