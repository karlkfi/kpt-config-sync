package differ

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/syncer/labeling"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDiff(t *testing.T) {
	testCases := []struct {
		name      string
		sync      v1alpha1.Sync
		lhs       runtime.Object
		rhs       runtime.Object
		wantEqual bool
		wantPanic bool
	}{
		{
			name: "resources with specified comparisons fields match",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "rbac.authorization.k8s.io",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Role",
									Versions: []v1alpha1.SyncVersion{
										{
											Version:       "v1",
											CompareFields: []string{"rules"},
										},
									},
								},
							},
						},
					},
				},
			},
			lhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
					{
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
				},
			},
			rhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
					{
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
				},
			},
			wantEqual: true,
		},
		{
			name: "resources with specified field comparisons don't match",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "rbac.authorization.k8s.io",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Role",
									Versions: []v1alpha1.SyncVersion{
										{
											Version:       "v1",
											CompareFields: []string{"rules"},
										},
									},
								},
							},
						},
					},
				},
			},
			lhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			rhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"list"},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "resource with spec fields matching",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "apps",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Deployment",
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
			lhs: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RecreateDeploymentStrategyType,
					},
				},
			},
			rhs: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RecreateDeploymentStrategyType,
					},
				},
			},
			wantEqual: true,
		},
		{
			name: "resource with spec fields not matching",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "apps",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Deployment",
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
			lhs: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RecreateDeploymentStrategyType,
					},
				},
			},
			rhs: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RollingUpdateDeploymentStrategyType,
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "resources with matching labels",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "rbac.authorization.k8s.io",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Role",
									Versions: []v1alpha1.SyncVersion{
										{
											Version:       "v1",
											CompareFields: []string{"rules"},
										},
									},
								},
							},
						},
					},
				},
			},
			lhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			rhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			wantEqual: true,
		},
		{
			name: "ignore management label when comparing",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "rbac.authorization.k8s.io",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Role",
									Versions: []v1alpha1.SyncVersion{
										{
											Version:       "v1",
											CompareFields: []string{"rules"},
										},
									},
								},
							},
						},
					},
				},
			},
			lhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						labeling.ResourceManagementKey: labeling.NomosSystemValue,
						"foo":                          "bar",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			rhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			wantEqual: true,
		},
		{
			name: "resources with annotations matching",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "rbac.authorization.k8s.io",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Role",
									Versions: []v1alpha1.SyncVersion{
										{
											Version:       "v1",
											CompareFields: []string{"rules"},
										},
									},
								},
							},
						},
					},
				},
			},
			lhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			rhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			wantEqual: true,
		},
		{
			name: "resources with annotations not matching",
			sync: v1alpha1.Sync{
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: "rbac.authorization.k8s.io",
							Kinds: []v1alpha1.SyncKind{
								{
									Kind: "Role",
									Versions: []v1alpha1.SyncVersion{
										{
											Version:       "v1",
											CompareFields: []string{"rules"},
										},
									},
								},
							},
						},
					},
				},
			},
			lhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			rhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"baz": "qux",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "clusterroles with different rules",
			lhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			rhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "readonly",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "clusterroles with different aggregation rules",
			lhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-edit": "true",
							},
						},
					},
				},
			},
			rhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "clusterroles with different aggregation rules, same rules",
			lhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-edit": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			rhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "clusterroles with same aggregation rules, same rules",
			lhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			rhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			wantEqual: true,
		},
		{
			name: "resources with different group, version, kinds",
			lhs: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
			},
			rhs: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
			},
			wantPanic: true,
		},
	}

	converter := runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if x := recover(); x != nil {
					if tc.wantPanic {
						return
					}
					panic(x)
				}
			}()

			differ := NewDiffer([]v1alpha1.Sync{tc.sync}, labeling.ResourceManagementKey)
			lhu, err := converter.ToUnstructured(tc.lhs)
			if err != nil {
				t.Fatalf("could not convert %v to unstructured type", tc.lhs)
			}
			rhu, err := converter.ToUnstructured(tc.rhs)
			if err != nil {
				t.Fatalf("could not convert %v to unstructured type", tc.rhs)
			}

			eq := differ.Equal(&unstructured.Unstructured{Object: lhu}, &unstructured.Unstructured{Object: rhu})
			if tc.wantPanic {
				t.Fatal("want panic, got none")
			}
			if eq != tc.wantEqual {
				t.Errorf("want Equal=%t, got %t", tc.wantEqual, eq)
			}
		})
	}
}
