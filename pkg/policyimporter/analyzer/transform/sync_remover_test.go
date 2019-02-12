package transform

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var syncRemoverTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewSyncRemover()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name: "single multiple gvk",
			Input: &ast.Root{
				System: &ast.System{
					Objects: vt.SystemObjectSets(&v1alpha1.Sync{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "nomos.dev/v1alpha1",
							Kind:       "Sync",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "bespin",
						},
						Spec: v1alpha1.SyncSpec{
							Groups: []v1alpha1.SyncGroup{
								{
									Group: "",
									Kinds: []v1alpha1.SyncKind{
										{
											Kind: "ResourceQuota",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
								{
									Group: "bespin.dev",
									Kinds: []v1alpha1.SyncKind{
										{
											Kind: "Folder",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
										{
											Kind: "Organization",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
										{
											Kind: "Project",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
										{
											Kind: "IAMPolicy",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
										{
											Kind: "ClusterIAMPolicy",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
										{
											Kind: "OrganizationPolicy",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
										{
											Kind: "ClusterOrganizationPolicy",
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
							},
						},
					},
					),
				},
			},
			ExpectOutput: &ast.Root{System: &ast.System{}},
		},
		{
			Name: "preserve hierarchy mode",
			Input: &ast.Root{
				System: &ast.System{
					Objects: vt.SystemObjectSets(&v1alpha1.Sync{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "nomos.dev/v1alpha1",
							Kind:       "Sync",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "bespin",
						},
						Spec: v1alpha1.SyncSpec{
							Groups: []v1alpha1.SyncGroup{
								{
									Group: "",
									Kinds: []v1alpha1.SyncKind{
										{
											Kind:          "ResourceQuota",
											HierarchyMode: v1alpha1.HierarchyModeHierarchicalQuota,
											Versions: []v1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
							},
						},
					},
					),
				},
			},
			ExpectOutput: &ast.Root{System: &ast.System{}},
		},
	},
}

func TestSyncRemover(t *testing.T) {
	syncRemoverTestcases.Run(t)
}
