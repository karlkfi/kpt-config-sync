/*
Copyright 2018 The Stolos Authors.
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

import (
	"fmt"
	"testing"
	"time"

	ph "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/actions"
	"github.com/google/nomos/pkg/syncer/flattening"
	"github.com/google/nomos/pkg/testing/rbactesting"
	"github.com/google/nomos/pkg/util/set/stringset"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	fakekube "k8s.io/client-go/kubernetes/fake"
)

// namedRoleBinding creates a test role for the given namespace with a given
// name.
func namedRoleBinding(namespace, name string) rbac.RoleBinding {
	return rbactesting.NamespaceRoleBinding(
		name, namespace, fmt.Sprintf("binding-from-%v", namespace), "User:joe")
}

// roleBinding mages a dummy role binding to be inserted into the given
// namespace.
func roleBinding(namespace string) rbac.RoleBinding {
	return namedRoleBinding(namespace, fmt.Sprintf("from-%v", namespace))
}

// policy creates a policy node with the given name. The created policy
// node is populated with dummy policies for testing.
func policy(name string) *flattening.Policy {
	return flattening.NewPolicy().
		AddRoleBinding(roleBinding(name)).
		AddRole(role(name))
}

// namedRole is same as namedRoleBinding, but for rbac.Role.
func namedRole(namespace, name string) rbac.Role {
	return rbactesting.NamespaceRole(
		name, namespace,
		[]rbac.PolicyRule{
			rbactesting.PolicyRule(
				[]string{"*"}, []string{"*"}, []string{"*"}),
		},
	)

}

// role is the same as roleBinding, but for rbac.Role.
func role(namespace string) rbac.Role {
	return namedRole(namespace, fmt.Sprintf("from-%v", namespace))
}

// policyNodeWithRole creates a policy node with the given name The created
// policy node is populated with a dummy policy for testing.
func policyWithRole(name string) *flattening.Policy {
	return flattening.NewPolicy().SetRoles(role(name))
}

type upsert struct {
	name   string
	parent string
	policy flattening.Policy
}

func TestPolicyEvaluatorRoleBindings(t *testing.T) {
	tests := []struct {
		name               string
		nodes              []upsert
		expectedNamespaces []string
		expectedPolicies   map[string]stringset.StringSet
	}{
		{
			name:               "all empty",
			nodes:              []upsert{},
			expectedNamespaces: []string{},
			expectedPolicies:   map[string]stringset.StringSet{},
		},
		{
			name: "no policies at all.",
			nodes: []upsert{
				upsert{"node", "", *flattening.NewPolicy()},
			},
			expectedNamespaces: []string{},
			expectedPolicies:   map[string]stringset.StringSet{},
		},
		{
			name: "One policy",
			nodes: []upsert{
				upsert{
					"node", "", *flattening.NewPolicy().SetRoleBindings(
						rbactesting.NamespaceRoleBinding(
							"role", "node", "pod-reader", "User:joe")),
				},
			},
			expectedNamespaces: []string{"node"},
			expectedPolicies: map[string]stringset.StringSet{
				"node": *stringset.NewFromValues("role"),
			},
		},
		{
			// The mini "acme" hierarchy.
			//
			// acme
			// + eng
			// | + frontend
			// | + backend
			// + prod
			name: "The mini acme hierarchy.",
			nodes: []upsert{
				upsert{"acme", "", *policy("acme")},
				upsert{"eng", "acme", *policy("eng")},
				upsert{"prod", "acme", *policy("prod")},
				upsert{"frontend", "eng", *policy("frontend")},
				upsert{"backend", "eng", *policy("backend")},
			},
			expectedNamespaces: []string{
				"acme", "eng", "prod", "frontend", "backend"},
			expectedPolicies: map[string]stringset.StringSet{
				"acme": *stringset.NewFromValues(
					"from-acme",
				),
				"eng": *stringset.NewFromValues(
					"from-acme", "from-eng",
				),
				"prod": *stringset.NewFromValues(
					"from-acme", "from-prod",
				),
				"frontend": *stringset.NewFromValues(
					"from-acme", "from-eng", "from-frontend",
				),
				"backend": *stringset.NewFromValues(
					"from-acme", "from-eng", "from-backend",
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := flattening.NewPolicyTree()
			for _, upsert := range tt.nodes {
				tree.Upsert(upsert.name, upsert.parent, false, upsert.policy)
			}
			e := newPolicyEvaluator(tree)
			for _, upsert := range tt.nodes {
				e.Visit(upsert.name, &upsert.policy)
			}
			ns := stringset.NewFromSlice(tt.expectedNamespaces)
			if !ns.Equals(e.RoleBindingNamespaces()) {
				t.Errorf("Missing namespaces: actual: %v, expected: %v",
					e.RoleBindingNamespaces(), ns)
			}
			for _, namespace := range tt.expectedNamespaces {
				expectedPolicy := tt.expectedPolicies[namespace]
				actualPolicy := roleBindingNames(e.RoleBindings(namespace))
				if !expectedPolicy.Equals(actualPolicy) {
					t.Errorf("Policy mismatch: ns=%v, actual: %v, expected: %v",
						namespace, actualPolicy, expectedPolicy)
				}
			}
		})
	}
}

func TestPolicyEvaluatorRoles(t *testing.T) {
	tests := []struct {
		name               string
		nodes              []upsert
		expectedNamespaces []string
		expectedPolicies   map[string]stringset.StringSet
	}{
		{
			name:               "all empty",
			nodes:              []upsert{},
			expectedNamespaces: []string{},
			expectedPolicies:   map[string]stringset.StringSet{},
		},
		{
			name: "no policies at all.",
			nodes: []upsert{
				upsert{"node", "", *flattening.NewPolicy()},
			},
			expectedNamespaces: []string{},
			expectedPolicies:   map[string]stringset.StringSet{},
		},
		{
			name: "One policy",
			nodes: []upsert{
				upsert{"node", "", *flattening.NewPolicy().SetRoles(
					rbactesting.NamespaceRole(
						"role", "node", []rbac.PolicyRule{}),
				)},
			},
			expectedNamespaces: []string{"node"},
			expectedPolicies: map[string]stringset.StringSet{
				"node": *stringset.NewFromValues("role"),
			},
		},
		{
			// The mini "acme" hierarchy.
			//
			// acme
			// + eng
			// | + frontend
			// | + backend
			// + prod
			name: "The mini acme hierarchy.",
			nodes: []upsert{
				upsert{"acme", "", *policyWithRole("acme")},
				upsert{"eng", "acme", *policyWithRole("eng")},
				upsert{"prod", "acme", *policyWithRole("prod")},
				upsert{"frontend", "eng", *policyWithRole("frontend")},
				upsert{"backend", "eng", *policyWithRole("backend")},
			},
			//nodes: []*flattening.PolicyNode{
			//acme, eng, frontend, backend, prod,
			//},
			expectedNamespaces: []string{
				"acme", "eng", "prod", "frontend", "backend"},
			expectedPolicies: map[string]stringset.StringSet{
				"acme": *stringset.NewFromValues(
					"from-acme",
				),
				"eng": *stringset.NewFromValues(
					"from-acme", "from-eng",
				),
				"prod": *stringset.NewFromValues(
					"from-acme", "from-prod",
				),
				"frontend": *stringset.NewFromValues(
					"from-acme", "from-eng", "from-frontend",
				),
				"backend": *stringset.NewFromValues(
					"from-acme", "from-eng", "from-backend",
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := flattening.NewPolicyTree()
			for _, upsert := range tt.nodes {
				tree.Upsert(upsert.name, upsert.parent, false, upsert.policy)
			}
			e := newPolicyEvaluator(tree)
			for _, node := range tt.nodes {
				e.Visit(node.name, &node.policy)
			}
			ns := stringset.NewFromSlice(tt.expectedNamespaces)
			if !ns.Equals(e.RoleNamespaces()) {
				t.Errorf("Missing namespaces: actual: %v, expected: %v",
					e.RoleBindingNamespaces(), ns)
			}
			for _, namespace := range tt.expectedNamespaces {
				expectedPolicy := tt.expectedPolicies[namespace]
				actualPolicy := roleNames(e.Roles(namespace))
				if !expectedPolicy.Equals(actualPolicy) {
					t.Errorf("Policy mismatch: ns=%v, actual: %v, expected: %v",
						namespace, actualPolicy, expectedPolicy)
				}
			}
		})
	}
}

func phPolicyNodeWithPolicyContent(name, parent string, policyspace bool, roles []rbac.Role,
	roleBindings []rbac.RoleBinding) *ph.PolicyNode {
	return rbactesting.PolicyNode(name, parent, policyspace, roles, roleBindings)
}

func phPolicyNode(name, parent string) *ph.PolicyNode {
	return phPolicyNodeWithPolicyContent(name, parent, false, []rbac.Role{},
		[]rbac.RoleBinding{roleBinding(name)})
}

func phPolicyNodePolicyspace(name, parent string) *ph.PolicyNode {
	return phPolicyNodeWithPolicyContent(name, parent, true, []rbac.Role{},
		[]rbac.RoleBinding{roleBinding(name)})
}

func phPolicyNodeWithRole(name, parent string) *ph.PolicyNode {
	return phPolicyNodeWithPolicyContent(name, parent, false,
		[]rbac.Role{role(name)},
		[]rbac.RoleBinding{})
}

func phPolicyNodeWithRolePolicyspace(name, parent string) *ph.PolicyNode {
	return phPolicyNodeWithPolicyContent(name, parent, true,
		[]rbac.Role{role(name)},
		[]rbac.RoleBinding{})
}

func TestFlatteningSyncerRoleBindings(t *testing.T) {
	acmeRbac := roleBinding("acme")
	unownedEngRbac := namedRoleBinding("eng", "some-unrelated-rolebinding")
	unknownRbac := roleBinding("unknown")
	type syncerFunc func(f *FlatteningSyncer)
	tests := []struct {
		name    string
		storage []runtime.Object
		// Syncer actions to be executed.
		actions         []syncerFunc
		expectedActions []string
	}{
		{
			name:            "Nothing",
			storage:         []runtime.Object{},
			expectedActions: []string{},
		},
		{
			name:    "One create.",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNode("acme", ""))
				},
			},
			expectedActions: []string{"rolebinding.acme.from-acme.upsert"},
		},
		{
			name:    "Mini acme create",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodePolicyspace("acme", ""))
					f.OnCreate(phPolicyNode("eng", "acme"))
				},
			},
			expectedActions: []string{
				"rolebinding.eng.from-acme.upsert",
				"rolebinding.eng.from-eng.upsert",
			},
		},
		{
			name:    "Acme create",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodePolicyspace("acme", ""))
					f.OnCreate(phPolicyNode("eng", "acme"))
					f.OnCreate(phPolicyNode("prod", "acme"))
					f.OnCreate(phPolicyNode("frontend", "eng"))
					f.OnCreate(phPolicyNode("backend", "eng"))
				},
			},
			expectedActions: []string{
				"rolebinding.eng.from-acme.upsert",
				"rolebinding.eng.from-eng.upsert",
				"rolebinding.prod.from-acme.upsert",
				"rolebinding.prod.from-prod.upsert",
				"rolebinding.frontend.from-acme.upsert",
				"rolebinding.frontend.from-eng.upsert",
				"rolebinding.frontend.from-frontend.upsert",
				"rolebinding.backend.from-acme.upsert",
				"rolebinding.backend.from-eng.upsert",
				"rolebinding.backend.from-backend.upsert",
			},
		},
		{
			name:    "Acme update",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodePolicyspace("acme", ""))
					f.OnCreate(phPolicyNode("eng", "acme"))
					f.OnCreate(phPolicyNode("frontend", "eng"))
					f.OnCreate(phPolicyNode("backend", "eng"))
					f.OnCreate(phPolicyNodePolicyspace("prod", "acme"))
					f.OnCreate(phPolicyNode("prj", "prod"))
				},
				func(f *FlatteningSyncer) {
					// Reparent "prj" from "prod" to "frontend".
					f.OnUpdate(
						phPolicyNode("prj", "prod"),
						phPolicyNode("prj", "frontend"))
				},
			},
			expectedActions: []string{
				"rolebinding.eng.from-acme.upsert",
				"rolebinding.eng.from-eng.upsert",
				"rolebinding.frontend.from-acme.upsert",
				"rolebinding.frontend.from-eng.upsert",
				"rolebinding.frontend.from-frontend.upsert",
				"rolebinding.backend.from-acme.upsert",
				"rolebinding.backend.from-eng.upsert",
				"rolebinding.backend.from-backend.upsert",
				"rolebinding.prj.from-acme.upsert",
				"rolebinding.prj.from-prod.upsert",
				"rolebinding.prj.from-prj.upsert",
				"rolebinding.prj.from-prod.delete",
				// Repeated inserts, perhaps inefficient.
				"rolebinding.prj.from-acme.upsert",
				"rolebinding.prj.from-eng.upsert",
				"rolebinding.prj.from-frontend.upsert",
				"rolebinding.prj.from-prj.upsert",
			},
		},
		{
			name:    "Acme delete",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNode("acme", ""))
					f.OnCreate(phPolicyNode("eng", "acme"))
					f.OnCreate(phPolicyNode("frontend", "eng"))
					f.OnCreate(phPolicyNode("backend", "eng"))
					f.OnCreate(phPolicyNode("prod", "acme"))
					f.OnCreate(phPolicyNode("prj", "prod"))
				},
				func(f *FlatteningSyncer) {
					// Declared parent node does not matter on delete, the
					// code consults the actual current parent.
					f.OnDelete(phPolicyNode("eng", ""))
				},
			},
			expectedActions: []string{
				"rolebinding.acme.from-acme.upsert", // 0
				"rolebinding.eng.from-acme.upsert",
				"rolebinding.eng.from-eng.upsert",
				"rolebinding.frontend.from-acme.upsert",
				"rolebinding.frontend.from-eng.upsert",
				"rolebinding.frontend.from-frontend.upsert", // 5
				"rolebinding.backend.from-acme.upsert",
				"rolebinding.backend.from-eng.upsert",
				"rolebinding.backend.from-backend.upsert",
				"rolebinding.prod.from-acme.upsert",
				"rolebinding.prod.from-prod.upsert", // 10
				"rolebinding.prj.from-acme.upsert",
				"rolebinding.prj.from-prod.upsert",
				"rolebinding.prj.from-prj.upsert",
				// Deleting the node "eng" removes "eng" and all the policies
				// in it, and all the policies in its child nodes.
				"rolebinding.backend.from-acme.delete",
				"rolebinding.backend.from-eng.delete",
				"rolebinding.backend.from-backend.delete",
				"rolebinding.eng.from-acme.delete",
				"rolebinding.eng.from-eng.delete",
				"rolebinding.frontend.from-acme.delete",
				"rolebinding.frontend.from-eng.delete",
				"rolebinding.frontend.from-frontend.delete",
			},
		},
		{
			name:    "PeriodicResync with empty policy node tree.",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.PeriodicResync([]*ph.PolicyNode{})
				},
			},
			expectedActions: []string{},
		},
		{
			name:    "Periodic Resync from scratch",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					if err := f.PeriodicResync([]*ph.PolicyNode{
						phPolicyNode("acme", ""),
						phPolicyNode("eng", "acme"),
						phPolicyNode("frontend", "eng"),
					}); err != nil {
						panic(err)
					}
				},
			},
			expectedActions: []string{
				"rolebinding.acme.from-acme.upsert",
				"rolebinding.eng.from-acme.upsert",
				"rolebinding.eng.from-eng.upsert",
				"rolebinding.frontend.from-acme.upsert",
				"rolebinding.frontend.from-eng.upsert",
				"rolebinding.frontend.from-frontend.upsert",
			},
		},
		{
			name: "Periodic Resync from partial content",
			storage: []runtime.Object{
				// unknownRbac is a policy in a namespace that is not managed
				// by stolos.   We don't touch such a policy.  But, there is a
				// policy unownedEngRbac that isn't mentioned in the refresh,
				// and that one we delete.
				&acmeRbac, &unknownRbac, &unownedEngRbac,
			},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					if err := f.PeriodicResync([]*ph.PolicyNode{
						phPolicyNode("eng", "acme"),
						phPolicyNode("frontend", "eng"),
					}); err != nil {
						panic(err)
					}
				},
			},
			expectedActions: []string{
				"rolebinding.eng.some-unrelated-rolebinding.delete",
				"rolebinding.eng.from-eng.upsert",
				"rolebinding.frontend.from-eng.upsert",
				"rolebinding.frontend.from-frontend.upsert",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recordingQueue := &testQueue{}
			fake := fakekube.NewSimpleClientset(tt.storage...)
			informerFactory :=
				informers.NewSharedInformerFactory(fake, 24*time.Hour)
			rbac := informerFactory.Rbac().V1()
			lister := rbac.RoleBindings().Lister()
			roleBindingAction := actions.NewRoleBindingResource(fake, lister)
			roleLister := rbac.Roles().Lister()
			roleActions := actions.NewRoleResource(fake, roleLister)
			syncer := NewFlatteningSyncer(
				recordingQueue, roleBindingAction, roleActions)
			informerFactory.Start(nil)
			informerFactory.WaitForCacheSync(nil)
			for _, op := range tt.actions {
				op(syncer)
			}
			CheckQueueActions(t, recordingQueue.items, tt.expectedActions)
		})
	}
}

func TestFlatteningSyncerRoles(t *testing.T) {
	acmeRbac := role("acme")
	unownedEngRbac := namedRole("eng", "some-unrelated-rolebinding")
	unknownRbac := role("unknown")
	type syncerFunc func(f *FlatteningSyncer)
	tests := []struct {
		name    string
		storage []runtime.Object
		// Syncer actions to be executed.
		actions         []syncerFunc
		expectedActions []string
	}{
		{
			name:            "Nothing",
			storage:         []runtime.Object{},
			expectedActions: []string{},
		},
		{
			name:    "One create.",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodeWithRole("acme", ""))
				},
			},
			expectedActions: []string{"role.acme.from-acme.upsert"},
		},
		{
			name:    "Add-delete-add",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodeWithRole("acme", ""))
					f.OnDelete(phPolicyNodeWithRole("acme", ""))
				},
				func(f *FlatteningSyncer) {
					if _, err := f.policyTree.Lookup("acme"); err == nil {
						t.Errorf("Unexpected policy for node 'acme'")
					}
				},
			},
			expectedActions: []string{
				"role.acme.from-acme.upsert",
				"role.acme.from-acme.delete",
			},
		},
		{
			name:    "Mini acme create",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodeWithRolePolicyspace("acme", ""))
					f.OnCreate(phPolicyNodeWithRole("eng", "acme"))
				},
			},
			expectedActions: []string{
				"role.eng.from-acme.upsert",
				"role.eng.from-eng.upsert",
			},
		},
		{
			name:    "Acme create",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodeWithRolePolicyspace("acme", ""))
					f.OnCreate(phPolicyNodeWithRole("eng", "acme"))
					f.OnCreate(phPolicyNodeWithRolePolicyspace("prod", "acme"))
					f.OnCreate(phPolicyNodeWithRole("frontend", "eng"))
					f.OnCreate(phPolicyNodeWithRole("backend", "eng"))
				},
			},
			expectedActions: []string{
				"role.eng.from-acme.upsert",
				"role.eng.from-eng.upsert",
				"role.frontend.from-acme.upsert",
				"role.frontend.from-eng.upsert",
				"role.frontend.from-frontend.upsert",
				"role.backend.from-acme.upsert",
				"role.backend.from-eng.upsert",
				"role.backend.from-backend.upsert",
			},
		},
		{
			name:    "Acme update",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodeWithRole("acme", ""))
					f.OnCreate(phPolicyNodeWithRole("eng", "acme"))
					f.OnCreate(phPolicyNodeWithRole("frontend", "eng"))
					f.OnCreate(phPolicyNodeWithRole("backend", "eng"))
					f.OnCreate(phPolicyNodeWithRole("prod", "acme"))
					f.OnCreate(phPolicyNodeWithRole("prj", "prod"))
				},
				func(f *FlatteningSyncer) {
					// Reparent "prj" from "prod" to "frontend".
					f.OnUpdate(
						phPolicyNodeWithRole("prj", "prod"),
						phPolicyNodeWithRole("prj", "frontend"))
				},
			},
			expectedActions: []string{
				"role.acme.from-acme.upsert",
				"role.eng.from-acme.upsert",
				"role.eng.from-eng.upsert",
				"role.frontend.from-acme.upsert",
				"role.frontend.from-eng.upsert",
				"role.frontend.from-frontend.upsert",
				"role.backend.from-acme.upsert",
				"role.backend.from-eng.upsert",
				"role.backend.from-backend.upsert",
				"role.prod.from-acme.upsert",
				"role.prod.from-prod.upsert",
				"role.prj.from-acme.upsert",
				"role.prj.from-prod.upsert",
				"role.prj.from-prj.upsert",
				"role.prj.from-prod.delete",
				// Repeated inserts, perhaps inefficient.
				"role.prj.from-acme.upsert",
				"role.prj.from-eng.upsert",
				"role.prj.from-frontend.upsert",
				"role.prj.from-prj.upsert",
			},
		},
		{
			name:    "Mini delete",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodeWithRolePolicyspace("acme", ""))
					f.OnCreate(phPolicyNodeWithRole("eng", "acme"))
					f.OnCreate(phPolicyNodeWithRole("frontend", "eng"))
				},
				func(f *FlatteningSyncer) {
					// Declared parent node does not matter on delete, the
					// code consults the actual current parent.
					f.OnDelete(phPolicyNodeWithRole("eng", ""))
				},
			},
			expectedActions: []string{
				"role.eng.from-acme.upsert",
				"role.eng.from-eng.upsert",
				"role.frontend.from-acme.upsert",
				"role.frontend.from-eng.upsert",
				"role.frontend.from-frontend.upsert", // 5
				"role.eng.from-acme.delete",
				"role.eng.from-eng.delete",
				"role.frontend.from-acme.delete",
				"role.frontend.from-eng.delete",
				"role.frontend.from-frontend.delete",
			},
		},
		{
			name:    "Acme delete",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.OnCreate(phPolicyNodeWithRole("acme", ""))
					f.OnCreate(phPolicyNodeWithRole("eng", "acme"))
					f.OnCreate(phPolicyNodeWithRole("frontend", "eng"))
					f.OnCreate(phPolicyNodeWithRole("backend", "eng"))
					f.OnCreate(phPolicyNodeWithRole("prod", "acme"))
					f.OnCreate(phPolicyNodeWithRole("prj", "prod"))
				},
				func(f *FlatteningSyncer) {
					// Declared parent node does not matter on delete, the
					// code consults the actual current parent.
					f.OnDelete(phPolicyNodeWithRole("eng", ""))
				},
			},
			expectedActions: []string{
				"role.acme.from-acme.upsert", // 0
				"role.eng.from-acme.upsert",
				"role.eng.from-eng.upsert",
				"role.frontend.from-acme.upsert",
				"role.frontend.from-eng.upsert",
				"role.frontend.from-frontend.upsert", // 5
				"role.backend.from-acme.upsert",
				"role.backend.from-eng.upsert",
				"role.backend.from-backend.upsert",
				"role.prod.from-acme.upsert",
				"role.prod.from-prod.upsert", // 10
				"role.prj.from-acme.upsert",
				"role.prj.from-prod.upsert",
				"role.prj.from-prj.upsert",
				// Deleting the node "eng" removes "eng" and all the policies
				// in it, and all the policies in its child nodes.
				"role.backend.from-acme.delete",
				"role.backend.from-eng.delete",
				"role.backend.from-backend.delete",
				"role.eng.from-acme.delete",
				"role.eng.from-eng.delete",
				"role.frontend.from-acme.delete",
				"role.frontend.from-eng.delete",
				"role.frontend.from-frontend.delete",
			},
		},
		{
			name:    "PeriodicResync with empty policy node tree.",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					f.PeriodicResync([]*ph.PolicyNode{})
				},
			},
			expectedActions: []string{},
		},
		{
			name:    "Periodic Resync from scratch",
			storage: []runtime.Object{},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					if err := f.PeriodicResync([]*ph.PolicyNode{
						phPolicyNodeWithRole("acme", ""),
						phPolicyNodeWithRole("eng", "acme"),
						phPolicyNodeWithRole("frontend", "eng"),
					}); err != nil {
						panic(err)
					}
				},
			},
			expectedActions: []string{
				"role.acme.from-acme.upsert",
				"role.eng.from-acme.upsert",
				"role.eng.from-eng.upsert",
				"role.frontend.from-acme.upsert",
				"role.frontend.from-eng.upsert",
				"role.frontend.from-frontend.upsert",
			},
		},
		{
			name: "Periodic Resync from partial content",
			storage: []runtime.Object{
				// unknownRbac is a policy in a namespace that is not managed
				// by stolos.   We don't touch such a policy.  But, there is a
				// policy unownedEngRbac that isn't mentioned in the refresh,
				// and that one we delete.
				&acmeRbac, &unknownRbac, &unownedEngRbac,
			},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					if err := f.PeriodicResync([]*ph.PolicyNode{
						phPolicyNodeWithRole("eng", "acme"),
						phPolicyNodeWithRole("frontend", "eng"),
					}); err != nil {
						panic(err)
					}
				},
			},
			expectedActions: []string{
				"role.eng.some-unrelated-rolebinding.delete",
				"role.eng.from-eng.upsert",
				"role.frontend.from-eng.upsert",
				"role.frontend.from-frontend.upsert",
			},
		},
		{
			name:    "Periodic sync removes nothing when policy node set gets smaller",
			storage: []runtime.Object{&acmeRbac},
			actions: []syncerFunc{
				func(f *FlatteningSyncer) {
					if err := f.PeriodicResync([]*ph.PolicyNode{}); err != nil {
						panic(err)
					}
				},
			},
			expectedActions: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recordingQueue := &testQueue{}
			fake := fakekube.NewSimpleClientset(tt.storage...)
			informerFactory :=
				informers.NewSharedInformerFactory(fake, 24*time.Hour)
			rbac := informerFactory.Rbac().V1()
			lister := rbac.RoleBindings().Lister()
			roleBindingAction := actions.NewRoleBindingResource(fake, lister)
			roleLister := rbac.Roles().Lister()
			roleActions := actions.NewRoleResource(fake, roleLister)
			syncer := NewFlatteningSyncer(
				recordingQueue, roleBindingAction, roleActions)
			informerFactory.Start(nil)
			informerFactory.WaitForCacheSync(nil)
			for _, op := range tt.actions {
				op(syncer)
			}
			CheckQueueActions(t, recordingQueue.items, tt.expectedActions)
		})
	}
}
