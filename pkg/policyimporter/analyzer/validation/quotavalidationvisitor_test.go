package validation

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeQuota(includeScopes bool, includeScopeSelector bool) *corev1.ResourceQuota {
	rq := &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "quota",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5"),
			},
		},
	}

	if includeScopes {
		rq.Spec.Scopes = []corev1.ResourceQuotaScope{corev1.ResourceQuotaScopeBestEffort}
	}

	if includeScopeSelector {
		rq.Spec.ScopeSelector = &corev1.ScopeSelector{
			MatchExpressions: []corev1.ScopedResourceSelectorRequirement{

				{ScopeName: corev1.ResourceQuotaScopePriorityClass,
					Operator: corev1.ScopeSelectorOpIn,
				},
			},
		}

	}
	return rq
}

func makeTree(rq *corev1.ResourceQuota) *ast.Root {
	return &ast.Root{
		Tree: &ast.TreeNode{
			Type:    node.AbstractNamespace,
			Path:    nomospath.FromSlash("namespaces"),
			Objects: vt.ObjectSets(vt.Helper.AcmeResourceQuota()),
			Children: []*ast.TreeNode{
				{
					Type: node.AbstractNamespace,
					Path: nomospath.FromSlash("namespaces/eng"),
					Children: []*ast.TreeNode{
						{
							Type: node.Namespace,
							Path: nomospath.FromSlash("namespaces/eng/frontend"),
							Objects: vt.ObjectSets(
								rq,
							),
						},
					},
				},
			},
		},
	}
}

type quotaValidationVisitorTestCase struct {
	name  string
	input *ast.Root
	error []string
}

var testcases = []quotaValidationVisitorTestCase{
	{
		name:  "no scoping",
		input: makeTree(makeQuota(false, false)),
	},
	{
		name:  "set Scopes",
		input: makeTree(makeQuota(true, false)),
		error: []string{vet.IllegalResourceQuotaFieldErrorCode},
	},
	{
		name:  "set ScopeSelector",
		input: makeTree(makeQuota(false, true)),
		error: []string{vet.IllegalResourceQuotaFieldErrorCode},
	},
	{
		name:  "set Scopes and ScopeSelector",
		input: makeTree(makeQuota(true, true)),
		error: []string{vet.IllegalResourceQuotaFieldErrorCode, vet.IllegalResourceQuotaFieldErrorCode},
	},
}

func (tc quotaValidationVisitorTestCase) Run(t *testing.T) {
	visitor := NewQuotaValidator()
	actual := tc.input.Accept(visitor)

	if tc.input != actual {
		t.Fatalf("expected noop, mismatch on expected vs actual: %s", cmp.Diff(tc.input, actual))
	}
	vettesting.ExpectErrors(tc.error, visitor.Error(), t)
}

func TestKnownResourceValidator(t *testing.T) {
	for _, tc := range testcases {
		t.Run(tc.name, tc.Run)
	}
}
