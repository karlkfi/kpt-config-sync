package kptfile

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func fakeEmptyResourceGroup(labels, annotations map[string]string) ast.FileObject {
	name := "test-rg"
	namespace := "test-namespace"
	rg := newResourceGroup(name, namespace, labels, annotations, nil)
	return ast.FileObject{
		Object: rg,
	}
}

func fakeResourceGroup(labels, annotations map[string]string) ast.FileObject {
	name := "test-rg"
	namespace := "test-namespace"
	ids := []ObjMetadata{
		{
			Name:      "default-name",
			Namespace: "",
			Group:     "apps",
			Kind:      "Deployment",
		},
		{
			Name:      "default-name",
			Namespace: "",
			Group:     "",
			Kind:      "ConfigMap",
		},
	}
	rg := newResourceGroup(name, namespace, labels, annotations, ids)
	return ast.FileObject{
		Object: rg,
	}
}

func fakeKptfile(labels, annotations map[string]string) ast.FileObject {
	kptfile := &Kptfile{}
	kptfile.SetName("name-not-important")
	kptfile.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kptGroup,
		Version: "v1alpha1",
		Kind:    kptKind,
	})
	kptfile.Inventory = Inventory{
		Identifier:  "test-rg",
		Namespace:   "test-namespace",
		Labels:      labels,
		Annotations: annotations,
	}
	return ast.FileObject{
		Object: kptfile,
	}
}

func fakeKptfileAtPath(path string) ast.FileObject {
	obj := fakeKptfile(nil, nil)
	obj.Relative = cmpath.RelativeOS(path)
	return obj
}

func TestGenerateResourceGroup(t *testing.T) {
	tcs := []struct {
		testName string
		input    []ast.FileObject
		want     []ast.FileObject
		err      error
	}{
		{
			testName: "no change when there is no kptfile found",
			input:    []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml")},
			want:     []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml")},
		},
		{
			testName: "empty ResourceGroup generated when there are no resources",
			input:    []ast.FileObject{fakeKptfile(nil, nil)},
			want:     []ast.FileObject{fakeEmptyResourceGroup(nil, nil)},
		},
		{
			testName: "ResourceGroup generated when there is one kptfile",
			input:    []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"), fakeKptfile(nil, nil)},
			want:     []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"), fakeResourceGroup(nil, nil)},
		},
		{
			testName: "ResourceGroup generated with random labels",
			input: []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeKptfile(map[string]string{"random-key": "random-value"}, nil)},
			want: []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeResourceGroup(map[string]string{"random-key": "random-value"}, nil)},
		},
		{
			testName: "ResourceGroup generated with random annotations",
			input: []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeKptfile(nil, map[string]string{"random-key": "random-value"})},
			want: []ast.FileObject{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeResourceGroup(nil, map[string]string{"random-key": "random-value"})},
		},
		{
			testName: "Multiple Kptfiles lead to an error",
			input:    []ast.FileObject{fakeKptfileAtPath("a/Kptfile"), fakeKptfileAtPath("b/Kptfile")},
			err:      fmt.Errorf("KNV1059: Repo must contain at most one Kptfile:\na/Kptfile\nb/Kptfile\n\nFor more information, see https://g.co/cloud/acm-errors#knv1059"),
		},
	}
	for _, tc := range tcs {
		actual, err := AsResourceGroup(tc.input)
		if tc.err == nil {
			if err != nil {
				t.Errorf("%s:\nunexpected error %v", tc.testName, err)
			}
			if diff := cmp.Diff(actual, tc.want, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("%s", diff)
			}
		} else {
			if err == nil {
				t.Errorf("expected error not happened")
			}
			if diff := cmp.Diff(err.Error(), tc.err.Error()); diff != "" {
				t.Errorf("%s", diff)
			}
		}
	}
}
