package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/kptfile"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
)

func fakeEmptyResourceGroup(labels, annotations map[string]string) core.Object {
	name := "test-rg"
	namespace := "test-namespace"
	return kptfile.NewResourceGroup(name, namespace, labels, annotations, nil)
}

func fakeResourceGroup(labels, annotations map[string]string) core.Object {
	name := "test-rg"
	namespace := "test-namespace"
	ids := []kptfile.ObjMetadata{
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
	return kptfile.NewResourceGroup(name, namespace, labels, annotations, ids)
}

func fakeKptfile(labels, annotations map[string]string) core.Object {
	obj := fake.KptFileObject()
	obj.Inventory = kptfile.Inventory{
		Identifier:  "test-rg",
		Namespace:   "test-namespace",
		Labels:      labels,
		Annotations: annotations,
	}
	return obj
}

func TestGenerateResourceGroup(t *testing.T) {
	tcs := []struct {
		testName string
		input    []core.Object
		want     []core.Object
		wantErr  error
	}{
		{
			testName: "no change when there is no kptfile found",
			input:    []core.Object{fake.Deployment("deployment")},
			want:     []core.Object{fake.Deployment("deployment")},
		},
		{
			testName: "empty ResourceGroup generated when there are no resources",
			input:    []core.Object{fakeKptfile(nil, nil)},
			want:     []core.Object{fakeEmptyResourceGroup(nil, nil)},
		},
		{
			testName: "ResourceGroup generated when there is one kptfile",
			input:    []core.Object{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"), fakeKptfile(nil, nil)},
			want:     []core.Object{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"), fakeResourceGroup(nil, nil)},
		},
		{
			testName: "ResourceGroup generated with random labels",
			input: []core.Object{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeKptfile(map[string]string{"random-key": "random-value"}, nil)},
			want: []core.Object{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeResourceGroup(map[string]string{"random-key": "random-value"}, nil)},
		},
		{
			testName: "ResourceGroup generated with random annotations",
			input: []core.Object{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeKptfile(nil, map[string]string{"random-key": "random-value"})},
			want: []core.Object{fake.Deployment("deployment"), fake.ConfigMapAtPath("cm.yaml"),
				fakeResourceGroup(nil, map[string]string{"random-key": "random-value"})},
		},
		{
			testName: "Multiple Kptfiles lead to an error",
			input:    []core.Object{fake.KptFileObject(core.Name("a")), fake.KptFileObject(core.Name("b"))},
			wantErr:  MultipleKptfilesError(fakeKptfile(nil, nil)),
		},
		{
			testName: "One Kptfile inventory without namespace",
			input:    []core.Object{fake.KptFileObject(inventoryNamespace(""))},
			wantErr:  InvalidKptfileError(".inventory.namespace shouldn't be empty", fake.KptFileObject(inventoryNamespace(""))),
		},
		{
			testName: "One Kptfile inventory without identifier",
			input:    []core.Object{fake.KptFileObject(inventoryIdentifier(""))},
			wantErr:  InvalidKptfileError(".inventory.identifier shouldn't be empty", fake.KptFileObject(inventoryIdentifier(""))),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			actual, err := AsResourceGroup(tc.input)
			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("got AsResourceGroup() = %v, want nil", err)
				}
				if diff := cmp.Diff(actual, tc.want, cmpopts.EquateEmpty()); diff != "" {
					t.Error(diff)
				}
			} else {
				if err == nil {
					t.Errorf("got AsResourceGroup() = nil, want %v", tc.wantErr)
				} else if !errors.Is(err, tc.wantErr) {
					t.Error(cmp.Diff(tc.wantErr, err))
				}
			}
		})
	}
}

func inventoryNamespace(namespace string) core.MetaMutator {
	return func(o core.Object) {
		if result, isKptfile := o.(*kptfile.Kptfile); isKptfile {
			result.Inventory.Namespace = namespace
		}
	}
}

func inventoryIdentifier(identifier string) core.MetaMutator {
	return func(o core.Object) {
		if result, isKptfile := o.(*kptfile.Kptfile); isKptfile {
			result.Inventory.Identifier = identifier
		}
	}
}
