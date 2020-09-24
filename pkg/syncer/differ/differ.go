// Package differ contains code for diffing sync-enabled resources, not
// necessarily known at compile time.
package differ

import (
	"fmt"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/lifecycle"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Type indicates the state of the given resource
type Type string

const (
	// NoOp indicates that no action should be taken.
	NoOp = Type("no-op")

	// Create indicates the resource should be created.
	Create = Type("create")

	// Update indicates the resource is declared and is on the API server, so we should
	// calculate a patch and apply it.
	Update = Type("update")

	// Delete indicates the resource should be deleted.
	Delete = Type("delete")

	// DeleteNsConfig indicates the namespaceconfig should be deleted.
	DeleteNsConfig = Type("deletensconfig")

	// Error indicates the resource's management annotation in the API server is invalid.
	Error = Type("error")

	// Unmanage indicates the resource's management annotation should be removed from the API Server.
	Unmanage = Type("unmanage")

	// UnmanageNamespace indicates that the resource is a special Namespace
	// which should not be deleted directly by ACM. It should be unmanaged instead.
	UnmanageNamespace = Type("unmanage-namespace")
)

// Diff is resource where Declared and Actual do not match.
type Diff struct {
	// Name is the name of the resource this diff is for.
	Name string
	// Declared is the resource as it exists in the repository.
	Declared *unstructured.Unstructured
	// Actual is the resource as it exists in the cluster.
	Actual *unstructured.Unstructured
}

// Type returns the type of the difference between the repository and the API Server.
func (d Diff) Type() Type {

	if d.Declared != nil {
		// The resource IS in the repository.
		if ManagementUnset(d.Declared) {
			// The declared resource has no resource management key, so it is managed.
			if d.Actual != nil {
				// The resource is also in the cluster, so update it.
				return Update
			}
			// The resource is not in the cluster, so create it.
			return Create
		}
		if ManagementDisabled(d.Declared) {
			// The resource is explicitly marked management disabled in the repository.
			if d.Actual != nil {
				if HasNomosMeta(d.Actual) {
					// Management is disabled for the resource, so remove the annotation from the API Server.
					return Unmanage
				}
			}
			// Management is disabled and there's no required changes to the resource.
			return NoOp
		}
		// The annotation in the repo is invalid, so show an error.
		return Error
	}

	// The resource IS NOT in the repository.
	if d.Actual != nil {
		// The resource IS on the API Server.
		if len(d.Actual.GetOwnerReferences()) > 0 {
			// We disallow deleting resources with owner references.
			// So, do not attempt to delete resources on cluster with this field specified.
			// Most likely the actual resource was generated by a controller and had the
			// owner object's managed annotations propagated down to it.
			//
			// For example, if a Deployment is managed with Nomos, it will generate a ReplicaSet
			// that will inherit all the Deployment's annotations, including the managed
			// annotations. We will ignore the ReplicaSet because the ReplicaSet will have its
			// OwnerReference field set.
			return NoOp
		}

		if !HasNomosMeta(d.Actual) {
			// No Nomos annotations or labels, so don't do anything.
			return NoOp
		}

		if ManagementEnabled(d.Actual) {
			// There are Nomos annotations or labels on the resource.

			if lifecycle.HasPreventDeletion(d.Actual) {
				// This object is marked with the lifecycle annotation that says to not
				// delete it. We should orphan the objects by unmanaging them.
				if d.Actual.GroupVersionKind().GroupKind() == kinds.Namespace().GroupKind() {
					// Special case for Namespaces. Not strictly necessary; but here to keep
					// consistent with namespace.go. Only happens in the Remediator, and
					// the Remediator doesn't do anything special for Namespaces.
					return UnmanageNamespace
				}
				return Unmanage
			}

			if IsManageableSystemNamespace(d.Actual) {
				// Don't delete this Namespace from the cluster; unmanage it.

				// The Syncer never creates a differ.Diff with a Namespace, so this only
				// happens in the Remediator.
				return UnmanageNamespace
			}

			// Delete resource with management enabled on API Server.
			return Delete
		}

		// The actual resource has Nomos artifacts and is explicitly unmanaged.
		return Unmanage
	}

	// The resource is neither on the API Server nor in the repo, so do nothing.
	return NoOp
}

// Diffs returns the diffs between declared and actual state. We generate a diff for each GroupVersionKind.
// The actual resources are for all versions of a GroupKind and the declared resources are for a particular GroupKind.
// We need to ensure there is not a declared resource across all possible versions before we delete it.
// The diffs will be returned in an arbitrary order.
func Diffs(declared []*unstructured.Unstructured, actuals []*unstructured.Unstructured, allDeclaredVersions map[string]bool) []*Diff {
	actualsMap := map[string]*unstructured.Unstructured{}
	for _, obj := range actuals {
		// Assume no collisions among resources on API Server.
		actualsMap[obj.GetName()] = obj
	}

	decls := asComparableMap(declared)
	var diffs []*Diff
	for name, decl := range decls {
		diffs = append(diffs, &Diff{
			Name:     name,
			Actual:   actualsMap[name],
			Declared: decl,
		})
	}
	for name, actual := range actualsMap {
		if !allDeclaredVersions[name] {
			// Not in any declared version, but on the API Server.
			diffs = append(diffs, &Diff{
				Name:   name,
				Actual: actual,
			})
		}
	}

	return diffs
}

func asComparableMap(us []*unstructured.Unstructured) map[string]*unstructured.Unstructured {
	m := map[string]*unstructured.Unstructured{}
	for _, u := range us {
		// Sometimes the status field is an empty struct. We disallow specifying the status field for configs,
		// so we should just remove anything set. This keeps us from making changes that will immediately get reverted by
		// other controllers.
		unstructured.RemoveNestedField(u.UnstructuredContent(), "status")
		name := u.GetName()
		if _, found := m[name]; found {
			panic(invalidInput{desc: fmt.Sprintf("Got duplicate decl for %q", name)})
		}
		m[name] = u
	}
	return m
}

type invalidInput struct {
	desc string
}

func (i *invalidInput) String() string {
	return i.desc
}
