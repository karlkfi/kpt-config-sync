package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type duplicateNameTestCase struct {
	name  string
	metas []ResourceMeta
	error []string
}

var duplicateNameTestCases = []duplicateNameTestCase{
	{
		name: "empty",
	},
	{
		name: "one resource",
		metas: []ResourceMeta{
			resourceMeta{name: "rb", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb.yaml"},
		},
	},
	{
		name: "two resources name collision",
		metas: []ResourceMeta{
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb1.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb2.yaml"},
		},
		error: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		name: "three resources name collision",
		metas: []ResourceMeta{
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb1.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb2.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb3.yaml"},
		},
		error: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		name: "two resources different name",
		metas: []ResourceMeta{
			resourceMeta{name: "name-1", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb1.yaml"},
			resourceMeta{name: "name-2", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb2.yaml"},
		},
	},
	{
		name: "two resources different directory",
		metas: []ResourceMeta{
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/foo/rb1.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/bar/rb2.yaml"},
		},
	},
	{
		name: "two resources different GroupKind",
		metas: []ResourceMeta{
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb1.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.Role(), source: "namespaces/rb2.yaml"},
		},
	},
	{
		name: "two resources different Version",
		metas: []ResourceMeta{
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding().GroupKind().WithVersion("v1"), source: "namespaces/rb1.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding().GroupKind().WithVersion("v2"), source: "namespaces/rb2.yaml"},
		},
		error: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		name: "parent directory name collision",
		metas: []ResourceMeta{
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/rb1.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.RoleBinding(), source: "namespaces/bar/rb2.yaml"},
		},
		error: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		name: "parent directory name collision not possible for ResourceQuotas",
		metas: []ResourceMeta{
			resourceMeta{name: "name", groupVersionKind: kinds.ResourceQuota(), source: "namespaces/rq1.yaml"},
			resourceMeta{name: "name", groupVersionKind: kinds.ResourceQuota(), source: "namespaces/bar/rq2.yaml"},
		},
	},
}

func (tc duplicateNameTestCase) Run(t *testing.T) {
	eb := multierror.Builder{}
	DuplicateNameValidatorFactory{}.New(tc.metas).Validate(&eb)

	vettesting.ExpectErrors(tc.error, eb.Build(), t)
}

func TestDuplicateNameValidator(t *testing.T) {
	for _, tc := range duplicateNameTestCases {
		t.Run(tc.name, tc.Run)
	}
}

// resourceMeta is a minimal implementation of ResourceMeta for use in tests.
type resourceMeta struct {
	source           string
	name             string
	groupVersionKind schema.GroupVersionKind
	meta             metav1.Object
}

var _ ResourceMeta = resourceMeta{}

// RelativeSlashPath implements ResourceMeta
func (m resourceMeta) RelativeSlashPath() string { return m.source }

// Dir implements ResourceMeta
func (m resourceMeta) Dir() nomospath.Relative { return nomospath.NewFakeRelative(m.source).Dir() }

// Name implements ResourceMeta
func (m resourceMeta) Name() string { return m.name }

// GroupVersionKind implements ResourceMeta
func (m resourceMeta) GroupVersionKind() schema.GroupVersionKind { return m.groupVersionKind }

// MetaObject implements ResourceMeta
func (m resourceMeta) MetaObject() metav1.Object { return m.meta }
