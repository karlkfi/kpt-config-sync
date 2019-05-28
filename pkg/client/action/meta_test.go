package action

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObjectMetaTestCase struct {
	Name         string
	Left         metav1.ObjectMeta
	Right        metav1.ObjectMeta
	ExpectReturn bool
}

func newObjectMeta(labels map[string]string, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Labels:      labels,
		Annotations: annotations,
	}
}

var objectMetaSubsetTestcases = []ObjectMetaTestCase{
	{
		Name: "labels and annotations are both subsets",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: true,
	},
	{
		Name: "labels not subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	{
		Name: "annotations not subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k5": "v5"},
		),
		ExpectReturn: false,
	},
	{
		Name: "neither are subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k6": "v6"},
		),
		ExpectReturn: false,
	},
}

func TestObjectMetaSubset(t *testing.T) {
	for _, tc := range objectMetaSubsetTestcases {
		t.Run(tc.Name, func(t *testing.T) {
			set := &rbacv1.Role{ObjectMeta: tc.Left}
			subset := &rbacv1.Role{ObjectMeta: tc.Right}
			if ObjectMetaSubset(set, subset) != tc.ExpectReturn {
				t.Fatalf("Testcase Failure %v", tc)
			}
		})
	}
}

var objectMetaEqualTestcases = []ObjectMetaTestCase{
	{
		Name: "labels and annotations are qual",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		ExpectReturn: true,
	},
	{
		Name:         "labels and annotations not set",
		Left:         newObjectMeta(nil, nil),
		Right:        newObjectMeta(nil, nil),
		ExpectReturn: true,
	},
	{
		Name: "labels and annotations are subsets",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	{
		Name: "neither are subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k6": "v6"},
		),
		ExpectReturn: false,
	},
	{
		Name: "left labels and annotations not set",
		Left: newObjectMeta(nil, nil),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	{
		Name: "right labels and annotations not set",
		Left: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		Right:        newObjectMeta(nil, nil),
		ExpectReturn: false,
	},
}

func TestObjectMetaEqual(t *testing.T) {
	for _, tc := range objectMetaEqualTestcases {
		t.Run(tc.Name, func(t *testing.T) {
			left := &rbacv1.Role{ObjectMeta: tc.Left}
			right := &rbacv1.Role{ObjectMeta: tc.Right}
			if ObjectMetaEqual(left, right) != tc.ExpectReturn {
				t.Fatalf("Testcase Failure %v", tc)
			}
		})
	}
}
