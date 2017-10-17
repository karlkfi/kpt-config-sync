package authorizer

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	authz "k8s.io/api/authorization/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAuthorize(t *testing.T) {
	tt := []struct {
		storage  []runtime.Object
		request  authz.SubjectAccessReviewSpec
		expected authz.SubjectAccessReviewStatus
	}{
		{
			storage: []runtime.Object{
				&v1.PolicyNode{},
			},
			request: authz.SubjectAccessReviewSpec{},
			expected: authz.SubjectAccessReviewStatus{
				Allowed:         false,
				EvaluationError: "ResourceAttributes missing",
			},
		},
		{
			storage: []runtime.Object{
				&v1.PolicyNode{
					ObjectMeta: meta.ObjectMeta{
						Name: "kitties",
					},
				},
			},
			request: authz.SubjectAccessReviewSpec{
				ResourceAttributes: &authz.ResourceAttributes{
					Name:      "meowie",
					Namespace: "kitties",
				},
			},
			expected: authz.SubjectAccessReviewStatus{
				Allowed: true,
			},
		},
		{
			storage: []runtime.Object{
				&v1.PolicyNode{
					ObjectMeta: meta.ObjectMeta{
						Name: "kitties",
					},
				},
				&v1.PolicyNode{
					ObjectMeta: meta.ObjectMeta{
						Name: "ponies",
					},
				},
			},
			request: authz.SubjectAccessReviewSpec{
				ResourceAttributes: &authz.ResourceAttributes{
					Name:      "horsie",
					Namespace: "ponies",
				},
			},
			expected: authz.SubjectAccessReviewStatus{
				Allowed: true,
			},
		},
	}

	for i, ttt := range tt {
		a := New(NewTestInformer(ttt.storage...))
		actual := a.Authorize(&ttt.request)
		if !reflect.DeepEqual(*actual, ttt.expected) {
			t.Errorf("[%v] Expected:\n%v\n---\nActual:\n%+v",
				i, spew.Sdump(ttt.expected), spew.Sdump(*actual))
		}
	}
}
