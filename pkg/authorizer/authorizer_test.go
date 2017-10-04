package authorizer

import (
	"reflect"
	"testing"

	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/policyhierarchy/fake"
	authz "k8s.io/api/authorization/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
						Name: "kitties",
					},
				},
				&v1.PolicyNode{
					ObjectMeta: metav1.ObjectMeta{
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

	for _, ttt := range tt {
		fakeClientSet := fake.NewSimpleClientset(ttt.storage...)
		a := New(fakeClientSet.K8usV1())
		actual := a.Authorize(&ttt.request)
		if !reflect.DeepEqual(*actual, ttt.expected) {
			t.Errorf("Expected:\n%+v\n---\nActual:\n%+v", ttt.expected, *actual)
		}
	}
}
