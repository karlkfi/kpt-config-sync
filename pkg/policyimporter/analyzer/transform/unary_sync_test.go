package transform

import (
	"strings"
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSync(group, kind string) *v1alpha1.Sync {
	name := strings.ToLower(kind)
	if group != "" {
		name += "." + group
	}
	return &v1alpha1.Sync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Sync",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.SyncSpec{
			Groups: []v1alpha1.SyncGroup{
				{
					Group: group,
					Kinds: []v1alpha1.SyncKind{
						{
							Kind: kind,
						},
					},
				},
			},
		},
	}
}

var unarySyncTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewUnarySync()
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
			ExpectOutput: &ast.Root{
				System: &ast.System{
					Objects: vt.SystemObjectSets(
						makeSync("", "ResourceQuota"),
						makeSync("bespin.dev", "Folder"),
						makeSync("bespin.dev", "Organization"),
						makeSync("bespin.dev", "Project"),
						makeSync("bespin.dev", "IAMPolicy"),
						makeSync("bespin.dev", "ClusterIAMPolicy"),
						makeSync("bespin.dev", "OrganizationPolicy"),
						makeSync("bespin.dev", "ClusterOrganizationPolicy"),
					),
				},
			},
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
			ExpectOutput: &ast.Root{
				System: &ast.System{
					Objects: vt.SystemObjectSets(
						makeSync("", "ResourceQuota"),
					),
				},
			},
		},
	},
}

func TestUnarySync(t *testing.T) {
	unarySyncTestcases.Run(t)
}
