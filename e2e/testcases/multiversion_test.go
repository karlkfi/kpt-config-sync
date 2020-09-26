package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
)

func TestMultipleVersions_CustomResource(t *testing.T) {
	nt := nomostest.New(t)

	// Add the Anvil CRD.
	nt.Root.Add("acme/cluster/anvil-crd.yaml", anvilCRD())
	nt.Root.CommitAndPush("Adding Anvil CRD")
	nt.WaitForRepoSync()
	nt.RenewClient()

	// Add the v1 and v1beta1 Anvils and verify they are created.
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/anvilv1.yaml", anvilCR("v1", "first", 10))
	nt.Root.Add("acme/namespaces/foo/anvilv2.yaml", anvilCR("v2", "second", 100))
	nt.Root.CommitAndPush("Adding v1 and v2 Anvil CRs")
	nt.WaitForRepoSync()

	err := nt.Validate("first", "foo", anvilCR("v1", "", 0))
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("second", "foo", anvilCR("v2", "", 0))
	if err != nil {
		t.Fatal(err)
	}

	// Modify the v1 and v1beta1 Anvils and verify they are updated.
	nt.Root.Add("acme/namespaces/foo/anvilv1.yaml", anvilCR("v1", "first", 20))
	nt.Root.Add("acme/namespaces/foo/anvilv2.yaml", anvilCR("v2", "second", 200))
	nt.Root.CommitAndPush("Modifying v1 and v2 Anvil CRs")
	nt.WaitForRepoSync()

	err = nt.Validate("first", "foo", anvilCR("v1", "", 0))
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("second", "foo", anvilCR("v2", "", 0))
	if err != nil {
		t.Fatal(err)
	}
}

func anvilCRD() *apiextensionsv1beta1.CustomResourceDefinition {
	crd := fake.CustomResourceDefinitionV1Beta1Object(core.Name("anvils.acme.com"))
	crd.Spec.Group = "acme.com"
	crd.Spec.Names = apiextensionsv1beta1.CustomResourceDefinitionNames{
		Plural:   "anvils",
		Singular: "anvil",
		Kind:     "Anvil",
	}
	crd.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	crd.Spec.Versions = []apiextensionsv1beta1.CustomResourceDefinitionVersion{
		{
			Name:    "v1",
			Served:  true,
			Storage: false,
		},
		{
			Name:    "v2",
			Served:  true,
			Storage: true,
		},
	}
	crd.Spec.Validation = &apiextensionsv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
			Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
				"spec": {
					Type:     "object",
					Required: []string{"lbs"},
					Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
						"lbs": {
							Type:    "integer",
							Minimum: pointer.Float64Ptr(1.0),
							Maximum: pointer.Float64Ptr(9000.0),
						},
					},
				},
			},
		},
	}
	return crd
}

func anvilCR(version, name string, weight int64) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "acme.com",
		Version: version,
		Kind:    "Anvil",
	})
	if name != "" {
		u.SetName(name)
	}
	if weight != 0 {
		u.Object["spec"] = map[string]interface{}{
			"lbs": weight,
		}
	}
	return u
}

func TestMultipleVersions_RoleBinding(t *testing.T) {
	nt := nomostest.New(t)

	rbV1 := fake.RoleBindingObject(core.Name("v1user"))
	rbV1.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Role",
		Name:     "acme-admin",
	}
	rbV1.Subjects = append(rbV1.Subjects, rbacv1.Subject{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "User",
		Name:     "v1user@acme.com",
	})

	rbV1Beta1 := fake.RoleBindingV1Beta1Object(core.Name("v1beta1user"))
	rbV1Beta1.RoleRef = rbacv1beta1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Role",
		Name:     "acme-admin",
	}
	rbV1Beta1.Subjects = append(rbV1Beta1.Subjects, rbacv1beta1.Subject{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "User",
		Name:     "v1beta1user@acme.com",
	})

	// Add the v1 and v1beta1 RoleBindings and verify they are created.
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/rbv1.yaml", rbV1)
	nt.Root.Add("acme/namespaces/foo/rbv1beta1.yaml", rbV1Beta1)
	nt.Root.CommitAndPush("Adding v1 and v1beta1 RoleBindings")
	nt.WaitForRepoSync()

	err := nt.Validate("v1user", "foo", &rbacv1.RoleBinding{},
		hasV1Subjects("v1user@acme.com"))
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("v1beta1user", "foo", &rbacv1beta1.RoleBinding{},
		hasV1Beta1Subjects("v1beta1user@acme.com"))
	if err != nil {
		t.Fatal(err)
	}

	// Modify the v1 and v1beta1 RoleBindings and verify they are updated.
	rbV1.Subjects = append(rbV1.Subjects, rbacv1.Subject{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "User",
		Name:     "v1admin@acme.com",
	})
	rbV1Beta1.Subjects = append(rbV1Beta1.Subjects, rbacv1beta1.Subject{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "User",
		Name:     "v1beta1admin@acme.com",
	})

	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/rbv1.yaml", rbV1)
	nt.Root.Add("acme/namespaces/foo/rbv1beta1.yaml", rbV1Beta1)
	nt.Root.CommitAndPush("Modifying v1 and v1beta1 RoleBindings")
	nt.WaitForRepoSync()

	err = nt.Validate("v1user", "foo", &rbacv1.RoleBinding{},
		hasV1Subjects("v1user@acme.com", "v1admin@acme.com"))
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("v1beta1user", "foo", &rbacv1beta1.RoleBinding{},
		hasV1Beta1Subjects("v1beta1user@acme.com", "v1beta1admin@acme.com"))
	if err != nil {
		t.Fatal(err)
	}

	// Remove the v1beta1 RoleBinding and verify that only it is deleted.
	nt.Root.Remove("acme/namespaces/foo/rbv1beta1.yaml")
	nt.Root.CommitAndPush("Removing v1beta1 RoleBinding")
	nt.WaitForRepoSync()

	if err := nt.Validate("v1user", "foo", &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}
	if err := nt.ValidateNotFound("v1beta1user", "foo", &rbacv1beta1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	// Remove the v1 RoleBinding and verify that it is also deleted.
	nt.Root.Remove("acme/namespaces/foo/rbv1.yaml")
	nt.Root.CommitAndPush("Removing v1 RoleBinding")
	nt.WaitForRepoSync()

	if err := nt.ValidateNotFound("v1user", "foo", &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}
}

func hasV1Subjects(subjects ...string) func(o core.Object) error {
	return func(o core.Object) error {
		r, ok := o.(*rbacv1.RoleBinding)
		if !ok {
			return nomostest.WrongTypeErr(o, r)
		}
		if len(r.Subjects) != len(subjects) {
			return errors.Wrapf(nomostest.ErrFailedPredicate, "want %v subjects; got %v", subjects, r.Subjects)
		}

		found := make(map[string]bool)
		for _, subj := range r.Subjects {
			found[subj.Name] = true
		}
		for _, name := range subjects {
			if !found[name] {
				return errors.Wrapf(nomostest.ErrFailedPredicate, "missing subject %q", name)
			}
		}

		return nil
	}
}

func hasV1Beta1Subjects(subjects ...string) func(o core.Object) error {
	return func(o core.Object) error {
		r, ok := o.(*rbacv1beta1.RoleBinding)
		if !ok {
			return nomostest.WrongTypeErr(o, r)
		}
		if len(r.Subjects) != len(subjects) {
			return errors.Wrapf(nomostest.ErrFailedPredicate, "want %v subjects; got %v", subjects, r.Subjects)
		}

		found := make(map[string]bool)
		for _, subj := range r.Subjects {
			found[subj.Name] = true
		}
		for _, name := range subjects {
			if !found[name] {
				return errors.Wrapf(nomostest.ErrFailedPredicate, "missing subject %q", name)
			}
		}

		return nil
	}
}
