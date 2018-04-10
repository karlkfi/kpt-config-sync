/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package syncer

// A syncer that flattens the PolicyNode hierarchy before writing it out.

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	ph "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/actions"
	"github.com/google/nomos/pkg/syncer/flattening"
	"github.com/google/nomos/pkg/util/set/stringset"
	"github.com/pkg/errors"
	rbac "k8s.io/api/rbac/v1"
)

var _ flattening.Visitor = (*policyEvaluator)(nil)

type policyLookupInterface interface {
	// Eval returns the resulting flattened policy for the given namespace.
	Eval(namespace string) (flattening.Policy, error)
}

// policyEvaluator accumulates all policies seen in all nodes visited by
// repeated calls to Visit.
type policyEvaluator struct {
	t policyLookupInterface
	// The set of roleBindingNamespaces seen so far in evaluating the policy.
	roleBindingNamespaces *stringset.StringSet
	// The set of policies, per each namespace encountered during evaluation.
	roleBindingsByNamespace map[string][]rbac.RoleBinding
	// The set of role namespaces seen so far in evaluating the policy.
	roleNamespaces *stringset.StringSet
	// The set of roles, per each namespace, encountered during evaluation.
	rolesByNamespace map[string][]rbac.Role
}

func newPolicyEvaluator(t policyLookupInterface) *policyEvaluator {
	return &policyEvaluator{
		t,
		stringset.New(),
		map[string][]rbac.RoleBinding{},
		stringset.New(),
		map[string][]rbac.Role{}}
}

// Visit implements flattening.Visitor.
func (p *policyEvaluator) Visit(name string, policy *flattening.Policy) {
	// TODO(filmil): PERFORMANCE It should be possible to short-circuit some of
	// the work here by caching.
	partial, err := p.t.Eval(name)
	if err != nil {
		glog.V(10).Infof("No such node: %q: %v", name, err)
		return
	}
	roleBindings := partial.RoleBindings()
	if len(roleBindings) > 0 {
		p.roleBindingNamespaces.Add(name)
	}
	for _, item := range roleBindings {
		p.roleBindingsByNamespace[item.Namespace] =
			append(p.roleBindingsByNamespace[item.Namespace], item)
	}

	roles := partial.Roles()
	if len(roles) > 0 {
		p.roleNamespaces.Add(name)
	}
	for _, item := range roles {
		p.rolesByNamespace[item.Namespace] = append(
			p.rolesByNamespace[item.Namespace], item)
	}
}

// RoleBindings returns the list of accumulated policies for the provided
// namespace.
func (p *policyEvaluator) RoleBindings(namespace string) []rbac.RoleBinding {
	return p.roleBindingsByNamespace[namespace]
}

// Roles returns the list of acumulated roles in the provided namespace.
func (p *policyEvaluator) Roles(namespace string) []rbac.Role {
	return p.rolesByNamespace[namespace]
}

// RoleBindingNamespaces returns the set of role binding namespaces that have
// been mentioned in the policy items evaluated by this evaluator.
func (p policyEvaluator) RoleBindingNamespaces() *stringset.StringSet {
	return p.roleBindingNamespaces
}

// RoleNamespaces returns the set of role namespaces that have been mentioned
// in the policy items evaluated by this evaluator.
func (p *policyEvaluator) RoleNamespaces() *stringset.StringSet {
	if p == nil {
		return stringset.New()
	}
	return p.roleNamespaces
}

var _ PolicyNodeSyncerInterface = (*FlatteningSyncer)(nil)

// Flattening syncer is a controller that watches PolicyNode objects and
// produces unpacked flattened policy objects.
type FlatteningSyncer struct {
	// The in-memory view of the policy tree.
	policyTree *flattening.PolicyTree

	// Encapsulates operations on role binding resources.
	roleBindingAction *actions.RoleBindingResource

	// Encapsulates operations on role resources.
	roleAction *actions.RoleResource

	// Used to enqueue mutations to the role bindings.
	queue Enqueuer
}

// NewFlatteningSyncer instantiates a PolicyNodeSyncerInterface that flattens
// the namespace policies.  queue is the queue used to submit the resulting
// actions into.  Action is the resource action specific to role binding
// resources.
func NewFlatteningSyncer(
	queue Enqueuer, action *actions.RoleBindingResource,
	roleAction *actions.RoleResource) *FlatteningSyncer {
	return &FlatteningSyncer{flattening.NewPolicyTree(), action, roleAction, queue}
}

// toPolicy unpacks the contents of a PolicyNode into the node name, the parent
// name, whether it's a policyspace and the policy attached to the node.
func toPolicy(node *ph.PolicyNode) (name, parent string, isPolicyspace bool, policy *flattening.Policy) {
	name = node.ObjectMeta.Name
	parent = node.Spec.Parent
	isPolicyspace = node.Spec.Policyspace
	policy = flattening.NewPolicy().
		SetRoleBindings(node.Spec.Policies.RoleBindingsV1...).
		SetRoles(node.Spec.Policies.RolesV1...)
	return // named
}

// EvalSubtree computes all the policies in a subtree of t starting from the
// node with the given name.  Returns all the aggregated policies, or error
// if the node with given name could not be found.
func EvalSubtree(t *flattening.PolicyTree, name string) (*policyEvaluator, error) {
	e := newPolicyEvaluator(t)
	t.VisitSubtree(name, e)
	return e, nil
}

// OnCreate implements PolicyNodeSyncerInterface
func (f *FlatteningSyncer) OnCreate(node *ph.PolicyNode) error {
	name, parent, isPolicyspace, policy := toPolicy(node)
	f.policyTree.Upsert(name, parent, isPolicyspace, *policy)
	result, err := EvalSubtree(f.policyTree, name)
	if err != nil {
		return errors.Wrapf(
			err, "OnCreate: while computing policies to add: node=%q", name)
	}
	result.RoleBindingNamespaces().ForEach(func(namespace string) {
		policies := result.RoleBindings(namespace)
		for i, _ := range policies {
			item := &policies[i]
			isPolicyspace, err := f.policyTree.IsPolicyspace(item.Namespace)
			if err != nil {
				return
			}
			if !isPolicyspace {
				glog.V(10).Infof("Upsert item: %v.%v", item.Namespace, item.Name)
				f.queue.Add(actions.NewRoleBindingUpsertAction(item, f.roleBindingAction))
			}
		}
	})
	// TODO(filmil) DUPLICATION
	result.RoleNamespaces().ForEach(func(namespace string) {
		policies := result.Roles(namespace)
		for i, _ := range policies {
			item := &policies[i]
			isPolicyspace, err := f.policyTree.IsPolicyspace(item.Namespace)
			if err != nil {
				return
			}
			if !isPolicyspace {
				glog.V(10).Infof("Upsert item: %v.%v", item.Namespace, item.Name)
				f.queue.Add(actions.NewRoleUpsertAction(item, f.roleAction))
			}
		}
	})
	return err
}

// roleBindingNames extracts the roleBindingNames of the nodes in the specified
// rbac policy.
func roleBindingNames(policy []rbac.RoleBinding) *stringset.StringSet {
	result := stringset.New()
	for _, item := range policy {
		result.Add(item.Name)
	}
	return result
}

// roleBindingNames extracts the roleBindingNames of the nodes in the specified
// rbac policy.
func roleNames(policy []rbac.Role) *stringset.StringSet {
	// TODO(filmil): DUPLICATION See if this can be unified with the above code.
	result := stringset.New()
	for _, item := range policy {
		result.Add(item.Name)
	}
	return result
}

// OnUpdate implements PolicyNodeSyncerInterface
func (f *FlatteningSyncer) OnUpdate(older *ph.PolicyNode, newer *ph.PolicyNode) error {
	name, newParent, isPolicyspace, newPolicy := toPolicy(newer)
	oldName, _, _, _ := toPolicy(older)

	olderResult, err := EvalSubtree(f.policyTree, oldName)
	if err != nil {
		return errors.Wrapf(err, "while finding older policies: %v")
	}
	f.policyTree.Upsert(name, newParent, isPolicyspace, *newPolicy)
	newerResult, err := EvalSubtree(f.policyTree, name)
	if err != nil {
		return errors.Wrapf(err, "while finding newer policies: %v")
	}
	stringset.Union(
		olderResult.RoleBindingNamespaces(),
		newerResult.RoleBindingNamespaces()).ForEach(
		func(namespace string) {
			oldPolicies := olderResult.RoleBindings(namespace)
			newPolicies := newerResult.RoleBindings(namespace)

			newNames := roleBindingNames(newPolicies)
			for i, _ := range oldPolicies {
				oldItem := &oldPolicies[i]
				if newNames.Contains(oldItem.Name) {
					continue
				}
				f.queue.Add(actions.NewRoleBindingDeleteAction(
					oldItem, f.roleBindingAction))
			}
			for i, _ := range newPolicies {
				item := &newPolicies[i]
				isPolicyspace, err := f.policyTree.IsPolicyspace(item.Namespace)
				if err != nil {
					return
				}
				if !isPolicyspace {
					f.queue.Add(actions.NewRoleBindingUpsertAction(item, f.roleBindingAction))
				}
			}
		},
	)
	// TODO(filmil): DUPLICATION Remove if possible.
	stringset.Union(
		olderResult.RoleNamespaces(),
		newerResult.RoleNamespaces()).ForEach(
		func(namespace string) {
			oldPolicies := olderResult.Roles(namespace)
			newPolicies := newerResult.Roles(namespace)

			newNames := roleNames(newPolicies)
			for i, _ := range oldPolicies {
				oldItem := &oldPolicies[i]
				if newNames.Contains(oldItem.Name) {
					continue
				}
				f.queue.Add(actions.NewRoleDeleteAction(
					oldItem.Namespace, oldItem.Name, f.roleAction))
			}
			for i, _ := range newPolicies {
				item := &newPolicies[i]
				isPolicyspace, err := f.policyTree.IsPolicyspace(item.Namespace)
				if err != nil {
					return
				}
				if !isPolicyspace {
					f.queue.Add(actions.NewRoleUpsertAction(item, f.roleAction))
				}
			}
		},
	)
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (f *FlatteningSyncer) OnDelete(node *ph.PolicyNode) error {
	var err error
	name, _, _, _ := toPolicy(node)
	result, err := EvalSubtree(f.policyTree, name)
	if err != nil {
		// Can't find a node that we're trying to delete.
		return errors.Wrapf(err, "OnDelete: while computing policies")
	}
	f.policyTree.Delete(name)
	result.RoleBindingNamespaces().StableForEach(func(namespace string) {
		policies := result.RoleBindings(namespace)
		for i := range policies {
			f.queue.Add(
				actions.NewRoleBindingDeleteAction(&policies[i], f.roleBindingAction))
		}
	})
	// TODO(filmil): DUPLICATION Remove if possible.
	result.RoleNamespaces().StableForEach(func(namespace string) {
		policies := result.Roles(namespace)
		for i := range policies {
			role := &policies[i]
			f.queue.Add(
				actions.NewRoleDeleteAction(role.Namespace, role.Name, f.roleAction))
		}
	})
	return nil
}

func newPolicyTree(nodes []*ph.PolicyNode) *flattening.PolicyTree {
	tree := flattening.NewPolicyTree()
	for _, node := range nodes {
		name, parent, isPolicyspace, policy := toPolicy(node)
		tree.Upsert(name, parent, isPolicyspace, *policy)
	}
	return tree
}

// PeriodicResync implements PolicyNodeSyncerInterface.
func (f *FlatteningSyncer) PeriodicResync(nodes []*ph.PolicyNode) error {
	if glog.V(11) {
		glog.V(11).Infof("Nodes: %v", spew.Sdump(nodes))
	}
	// Rebuild the policy tree from the current view.
	policyTree := newPolicyTree(nodes)
	roots := policyTree.Roots()
	glog.V(10).Infof("Roots: %+v", roots)

	// The periodic resync will remove unpacked resources only from the
	// namespaces that are part of the current 'nodes' view.  If a namespace
	// used to be managed but isn't any more, it will not be touched by the
	// code below.
	var err error
	for _, root := range roots {
		result, err2 := EvalSubtree(policyTree, root)
		if err2 != nil {
			return errors.Wrapf(err2, "while evaluating root: %q", root)
		}

		result.RoleBindingNamespaces().StableForEach(func(namespace string) {
			glog.V(10).Infof("Processing namespace: %q", namespace)
			presentResources, err2 := f.roleBindingAction.Values(namespace)
			if err2 != nil {
				err = errors.Wrapf(err2, "while listing namespace: %q", namespace)
				return
			}
			// The list of policies from the resync.
			policies := result.RoleBindings(namespace)
			specified := stringset.New()
			for _, policy := range policies {
				glog.V(10).Infof("Adding: %v", policy.Name)
				specified.Add(policy.Name)
			}
			for name, item := range presentResources {
				if specified.Contains(name) {
					continue
				}
				f.queue.Add(
					actions.NewRoleBindingDeleteAction(item.(*rbac.RoleBinding), f.roleBindingAction))
			}
			for i := range policies {
				item := &policies[i]
				isPolicyspace, err := policyTree.IsPolicyspace(item.Namespace)
				if err != nil {
					return
				}
				if !isPolicyspace {
					f.queue.Add(
						actions.NewRoleBindingUpsertAction(item, f.roleBindingAction))
				}
			}
		})
		// TODO(filmil): DUPLICATION Remove if possible.
		result.RoleNamespaces().StableForEach(func(namespace string) {
			glog.V(10).Infof("Processing namespace: %q", namespace)
			presentResources, err2 := f.roleAction.Values(namespace)
			if err2 != nil {
				err = errors.Wrapf(err2, "while listing namespace: %q", namespace)
				return
			}
			// The list of policies policies from the resync.
			policies := result.Roles(namespace)
			specified := stringset.New()
			for _, policy := range policies {
				glog.V(10).Infof("Adding: %v", policy.Name)
				specified.Add(policy.Name)
			}
			for name, item := range presentResources {
				if specified.Contains(name) {
					continue
				}
				role := item.(*rbac.Role)
				f.queue.Add(
					actions.NewRoleDeleteAction(role.Namespace, role.Name, f.roleAction))
			}
			for i := range policies {
				item := &policies[i]
				isPolicyspace, err := policyTree.IsPolicyspace(item.Namespace)
				if err != nil {
					return
				}
				if !isPolicyspace {
					f.queue.Add(
						actions.NewRoleUpsertAction(item, f.roleAction))
				}
			}
		})
	}
	return err
}
