package resourcequota

import (
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type EqualsTestCase struct {
	left   v1.ResourceList
	right  v1.ResourceList
	equals bool
}

func TestEquals(t *testing.T) {
	for i, tt := range []EqualsTestCase{
		{
			left: v1.ResourceList{
				"hay":  resource.MustParse("5"),
				"milk": resource.MustParse("2.0"),
			},
			right: v1.ResourceList{
				"milk": resource.MustParse("2"),
				"hay":  resource.MustParse("5.0"),
			},
			equals: true,
		},
		{
			left: v1.ResourceList{
				"hay": resource.MustParse("5"),
			},
			right: v1.ResourceList{
				"hay": resource.MustParse("5.1"),
			},
			equals: false,
		},
		{
			left: v1.ResourceList{
				"hay": resource.MustParse("5"),
			},
			right: v1.ResourceList{
				"milk": resource.MustParse("5"),
			},
			equals: false,
		},
		{
			left: v1.ResourceList{
				"hay": resource.MustParse("5"),
			},
			right:  v1.ResourceList{},
			equals: false,
		},
	} {
		actualEquals := resourceListEqual(tt.left, tt.right)

		if actualEquals != tt.equals {
			t.Errorf("[%d]Expected equal=%t but wasn't [%s] vs [%s]", i, tt.equals, tt.left, tt.right)
		}
	}
}

type DiffTestCase struct {
	left  v1.ResourceList
	right v1.ResourceList
	diff  v1.ResourceList
}

func TestDiff(t *testing.T) {
	for i, tt := range []DiffTestCase{
		{
			left: v1.ResourceList{
				"hay":  resource.MustParse("5"),
				"milk": resource.MustParse("2.0"),
			},
			right: v1.ResourceList{
				// different order and different format.
				"milk": resource.MustParse("2"),
				"hay":  resource.MustParse("5.0"),
			},
			diff: v1.ResourceList{},
		},
		{
			left: v1.ResourceList{
				"hay":  resource.MustParse("5"),
				"milk": resource.MustParse("2.0"),
			},
			right: v1.ResourceList{
				"hay": resource.MustParse("5.0"),
			},
			diff: v1.ResourceList{
				"milk": resource.MustParse("-2"),
			},
		},
		{
			left: v1.ResourceList{
				"hay": resource.MustParse("5"),
			},
			right: v1.ResourceList{
				"hay":  resource.MustParse("5.0"),
				"milk": resource.MustParse("2.0"),
			},
			diff: v1.ResourceList{
				"milk": resource.MustParse("2"),
			},
		},
	} {
		actual := diffResourceLists(tt.left, tt.right)

		if !resourceListEqual(actual, tt.diff) {
			t.Errorf("[%d]Expected diff to be [%s] but got [%s]", i, actual, tt.diff)
		}
	}
}
