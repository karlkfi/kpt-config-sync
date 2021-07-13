package e2e

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/e2e/nomostest"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem/gatekeeper"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestResourceConditionAnnotations(t *testing.T) {
	nt := nomostest.New(t)

	if nt.MultiRepo {
		// TODO(b/168918861): Re-enable this test when multi-repo supports resource condition annotations.
		return
	}

	ns := "rc-annotations"
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", ns),
		fake.NamespaceObject(ns))

	crName := "e2e-test-clusterrole"
	cr := fake.ClusterRoleObject(core.Name(crName))
	cr.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"deployments"},
		Verbs:     []string{"get", "list"},
	}}
	nt.Root.Add("acme/cluster/cr.yaml", cr)

	cmName := "e2e-test-configmap"
	cm := fake.ConfigMapObject(core.Name(cmName))
	cmPath := fmt.Sprintf("acme/namespaces/%s/configmap.yaml", ns)
	nt.Root.Add(cmPath, cm)
	nt.Root.CommitAndPush("add ConfigMap and ClusterRole with no annotations")
	// The bats test checks the NamespaceConfig/ClusterConfig, but checking the Repo
	// is sufficient.
	nt.WaitForRepoSyncs()

	// Ensure we don't already have error conditions.
	// In this test, and so below, it is sufficient to block on the Repo object reporting
	// the conditions, as all it is doing is aggregating conditions from ClusterConfig/NamespaceConfigs.
	err1 := nt.Validate("repo", "", &v1.Repo{},
		hasConditions())
	err2 := nt.Validate(v1.ClusterConfigName, "", &v1.ClusterConfig{},
		hasConditions())
	err3 := nt.Validate(ns, "", &v1.NamespaceConfig{},
		hasConditions())
	if err1 != nil || err2 != nil || err3 != nil {
		// There isn't a concise way of saying "If one of these three conditions fail,
		// show all errors and then fail the test."
		if err1 != nil {
			nt.T.Error(err1)
		}
		if err2 != nil {
			nt.T.Error(err2)
		}
		if err3 != nil {
			nt.T.Error(err3)
		}
		t.FailNow()
	}

	// Test adding error annotations.
	nt.MustKubectl("annotate", "clusterrole", crName,
		`configmanagement.gke.io/errors=["CrashLoopBackOff"]`)
	nt.MustKubectl("annotate", "configmap", cmName, "-n", ns,
		`configmanagement.gke.io/errors=["CrashLoopBackOff"]`)

	// Ensure error conditions are added.
	_, err1 = nomostest.Retry(20*time.Second, func() error {
		// We expect three errors even though we only supplied two.
		return nt.Validate("repo", "", &v1.Repo{},
			hasConditions(string(v1.ResourceStateError), string(v1.ResourceStateError), string(v1.ResourceStateError)))
	})
	// The ClusterConfig error from the ClusterRole gets duplicated.
	// This will be obsolete with ConfigSync v2, so no need to fix (b/154226839).
	err2 = nt.Validate(v1.ClusterConfigName, "", &v1.ClusterConfig{},
		hasConditions(string(v1.ResourceStateError), string(v1.ResourceStateError)))
	err3 = nt.Validate(ns, "", &v1.NamespaceConfig{},
		hasConditions(string(v1.ResourceStateError)))
	if err1 != nil || err2 != nil || err3 != nil {
		if err1 != nil {
			nt.T.Error(err1)
		}
		if err2 != nil {
			nt.T.Error(err2)
		}
		if err3 != nil {
			nt.T.Error(err3)
		}
		t.FailNow()
	}

	// Test removing error annotations.
	nt.MustKubectl("annotate", "clusterrole", crName,
		`configmanagement.gke.io/errors-`)
	nt.MustKubectl("annotate", "configmap", cmName, "-n", ns,
		`configmanagement.gke.io/errors-`)

	// Ensure error conditions are removed.
	_, err1 = nomostest.Retry(20*time.Second, func() error {
		return nt.Validate("repo", "", &v1.Repo{},
			hasConditions())
	})
	err2 = nt.Validate(v1.ClusterConfigName, "", &v1.ClusterConfig{},
		hasConditions())
	err3 = nt.Validate(ns, "", &v1.NamespaceConfig{},
		hasConditions())
	if err1 != nil || err2 != nil || err3 != nil {
		// There isn't a concise way of saying "If one of these three conditions fail,
		// show all errors and then fail the test."
		if err1 != nil {
			nt.T.Error(err1)
		}
		if err2 != nil {
			nt.T.Error(err2)
		}
		if err3 != nil {
			nt.T.Error(err3)
		}
		t.FailNow()
	}

	// Test adding reconciling annotations
	nt.MustKubectl("annotate", "clusterrole", crName,
		`configmanagement.gke.io/reconciling=["ConfigMap is incomplete", "ConfigMap is not ready"]`)
	nt.MustKubectl("annotate", "configmap", cmName, "-n", ns,
		`configmanagement.gke.io/reconciling=["ClusterRole needs... something..."]`)

	// Ensure reconciling conditions are added.
	_, err1 = nomostest.Retry(20*time.Second, func() error {
		// We expect three reconciling conditions even though we only supplied two.
		return nt.Validate("repo", "", &v1.Repo{},
			hasConditions(string(v1.ResourceStateReconciling), string(v1.ResourceStateReconciling), string(v1.ResourceStateReconciling)))
	})
	// The ClusterConfig condition from the ClusterRole gets duplicated.
	// This will be obsolete with ConfigSync v2, so no need to fix (b/154226839).
	err2 = nt.Validate(v1.ClusterConfigName, "", &v1.ClusterConfig{},
		hasConditions(string(v1.ResourceStateReconciling), string(v1.ResourceStateReconciling)))
	err3 = nt.Validate(ns, "", &v1.NamespaceConfig{},
		hasConditions(string(v1.ResourceStateReconciling)))
	if err1 != nil || err2 != nil || err3 != nil {
		if err1 != nil {
			nt.T.Error(err1)
		}
		if err2 != nil {
			nt.T.Error(err2)
		}
		if err3 != nil {
			nt.T.Error(err3)
		}
		t.FailNow()
	}

	// Test removing reconciling annotations.
	nt.MustKubectl("annotate", "clusterrole", crName,
		`configmanagement.gke.io/reconciling-`)
	nt.MustKubectl("annotate", "configmap", cmName, "-n", ns,
		`configmanagement.gke.io/reconciling-`)

	// Ensure reconciling conditions are removed.
	_, err1 = nomostest.Retry(20*time.Second, func() error {
		return nt.Validate("repo", "", &v1.Repo{},
			hasConditions())
	})
	err2 = nt.Validate(v1.ClusterConfigName, "", &v1.ClusterConfig{},
		hasConditions())
	err3 = nt.Validate(ns, "", &v1.NamespaceConfig{},
		hasConditions())
	if err1 != nil || err2 != nil || err3 != nil {
		if err1 != nil {
			nt.T.Error(err1)
		}
		if err2 != nil {
			nt.T.Error(err2)
		}
		if err3 != nil {
			nt.T.Error(err3)
		}
		t.FailNow()
	}
}

func TestConstraintTemplateStatusAnnotations(t *testing.T) {
	nt := nomostest.New(t)

	if nt.MultiRepo {
		// TODO(b/168918861): Re-enable this test when multi-repo supports resource condition annotations.
		return
	}
	support, err := nt.SupportV1Beta1CRD()
	if err != nil {
		nt.T.Fatal("failed to check the supported CRD versions")
	}
	// Skip this test when the v1beta1 CRD is not supported in the testing cluster.
	if !support {
		return
	}

	if err := nt.ApplyGatekeeperTestData("constraint-template-crd.yaml", "constrainttemplates.templates.gatekeeper.sh"); err != nil {
		nt.T.Fatalf("Failed to create constraint template CRD: %v", err)
	}

	// Create and apply a ConstraintTemplate.
	ctName := "k8sname"
	ctGVK := schema.GroupVersionKind{
		Group:   gatekeeper.TemplatesGroup,
		Version: "v1beta1",
		Kind:    "ConstraintTemplate",
	}
	ct := fake.UnstructuredObject(ctGVK, core.Name(ctName))
	ct.Object["spec"] = map[string]interface{}{
		"crd": map[string]interface{}{
			"spec": map[string]interface{}{
				"names": map[string]interface{}{
					"kind": "K8sName",
				},
			},
		},
		"targets": []interface{}{
			map[string]interface{}{
				"target": "admission.k8s.gatekeeper.sh",
				"rego": `package k8sname
        violation[{"msg": msg}] {
          input.review.object.metadata.name == "policycontroller-violation"
          msg := "object is called policycontroller-violation"
        }`,
			},
		},
	}
	nt.Root.Add("acme/cluster/constraint-template.yaml", ct)
	nt.Root.CommitAndPush("add gatekeeper ConstraintTemplate")
	nt.WaitForRepoSyncs()

	// In the real world, this annotation would be removed once PolicyController
	// created the CRD corresponding to this ConstraintTemplate. Thus, this test
	// requires Gatekeeper to not be installed to test this path in a non-flaky way.
	_, err = nomostest.Retry(20*time.Second, func() error {
		// This happens asynchronously with syncing the repo; so the Repo may report
		// "synced" before this appears.
		return nt.Validate(ctName, "", fake.UnstructuredObject(ctGVK),
			nomostest.HasAnnotation(metadata.ResourceStatusReconcilingKey, `["ConstraintTemplate has not been created"]`))
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func TestConstraintStatusAnnotations(t *testing.T) {
	nt := nomostest.New(t)

	if nt.MultiRepo {
		// TODO(b/168918861): Re-enable this test when multi-repo supports resource condition annotations.
		return
	}
	support, err := nt.SupportV1Beta1CRD()
	if err != nil {
		nt.T.Fatal("failed to check the supported CRD versions")
	}
	// Skip this test when v1beta1 CRD is not supported in the testing cluster.
	if !support {
		return
	}

	if err := nt.ApplyGatekeeperTestData("constraint-crd.yaml", "k8sallowedrepos.constraints.gatekeeper.sh"); err != nil {
		nt.T.Fatalf("Failed to create constraint template CRD: %v", err)
	}

	constraintGVK := schema.GroupVersionKind{
		Group:   gatekeeper.ConstraintsGroup,
		Version: "v1beta1",
		Kind:    "K8sAllowedRepos",
	}
	constraintName := "prod-pod-is-fun"
	constraint := fake.UnstructuredObject(constraintGVK, core.Name(constraintName))
	constraint.Object["spec"] = map[string]interface{}{
		"match": map[string]interface{}{
			"kinds": []interface{}{
				map[string]interface{}{
					"apiGroups": []interface{}{""},
					"kinds":     []interface{}{"Pod"},
				},
			},
		},
		"parameters": map[string]interface{}{
			"repos": []interface{}{"only-this-repo"},
		},
	}
	nt.Root.Add("acme/cluster/constraint.yaml", constraint)
	nt.Root.CommitAndPush("Add Gatekeeper Constraint")
	nt.WaitForRepoSyncs()

	// In the real world, this annotation would be removed once PolicyController
	// began enforcing it. Thus, this test requires Gatekeeper to not be installed
	// to test this path in a non-flaky way.
	_, err = nomostest.Retry(20*time.Second, func() error {
		// This happens asynchronously with syncing the repo; so the Repo may report
		// "synced" before this appears.
		return nt.Validate(constraintName, "", fake.UnstructuredObject(constraintGVK),
			nomostest.HasAnnotation(metadata.ResourceStatusReconcilingKey, `["Constraint has not been processed by PolicyController"]`))
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func hasConditions(want ...string) nomostest.Predicate {
	sort.Strings(want)
	return func(o client.Object) error {
		var got []string
		switch obj := o.(type) {
		case *v1.NamespaceConfig:
			for _, rc := range obj.Status.ResourceConditions {
				got = append(got, string(rc.ResourceState))
			}
		case *v1.ClusterConfig:
			for _, rc := range obj.Status.ResourceConditions {
				got = append(got, string(rc.ResourceState))
			}
		case *v1.Repo:
			for _, rc := range obj.Status.Sync.ResourceConditions {
				got = append(got, string(rc.ResourceState))
			}
		default:
			return errors.Wrapf(nomostest.ErrWrongType, "got %T, expect one of (%T, %T, %T)",
				o, &v1.NamespaceConfig{}, &v1.ClusterConfig{}, &v1.Repo{})
		}
		if diff := cmp.Diff(want, got); diff != "" {
			return errors.Errorf("unexpected resource condition diff: %s", diff)
		}
		return nil
	}
}
